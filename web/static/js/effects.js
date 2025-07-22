/**
 * 视觉效果模块
 * 包含粒子系统、光晕效果、动画等炫酷效果
 */

class EffectsEngine {
    constructor(scene, renderer) {
        this.scene = scene;
        this.renderer = renderer;
        this.particles = [];
        this.connections = [];
        this.searchRipples = [];
        this.animationId = null;
        this.isAnimating = true;
        
        this.setupPostProcessing();
        this.createStarField();
    }

    /**
     * 设置后处理效果
     */
    setupPostProcessing() {
        // 基础设置，如果需要更高级的后处理效果可以扩展
        this.renderer.setPixelRatio(Math.min(window.devicePixelRatio, 2));
        this.renderer.setClearColor(0x0a0a0a, 1);
        this.renderer.shadowMap.enabled = true;
        this.renderer.shadowMap.type = THREE.PCFSoftShadowMap;
    }

    /**
     * 创建星空背景
     */
    createStarField() {
        const starGeometry = new THREE.BufferGeometry();
        const starCount = 2000;
        const positions = new Float32Array(starCount * 3);
        const colors = new Float32Array(starCount * 3);

        for (let i = 0; i < starCount; i++) {
            // 位置
            positions[i * 3] = (Math.random() - 0.5) * 200;
            positions[i * 3 + 1] = (Math.random() - 0.5) * 200;
            positions[i * 3 + 2] = (Math.random() - 0.5) * 200;

            // 颜色 (蓝白色调)
            const intensity = Math.random() * 0.5 + 0.5;
            colors[i * 3] = intensity * 0.8;     // R
            colors[i * 3 + 1] = intensity * 0.9; // G
            colors[i * 3 + 2] = intensity;       // B
        }

        starGeometry.setAttribute('position', new THREE.BufferAttribute(positions, 3));
        starGeometry.setAttribute('color', new THREE.BufferAttribute(colors, 3));

        const starMaterial = new THREE.PointsMaterial({
            size: 0.5,
            vertexColors: true,
            transparent: true,
            opacity: 0.6,
            sizeAttenuation: true
        });

        this.starField = new THREE.Points(starGeometry, starMaterial);
        this.scene.add(this.starField);
    }

    /**
     * 创建粒子效果 - 优化版本
     */
    createParticle(position, color = 0x4ecdc4, size = 1.2) {
        const geometry = new THREE.SphereGeometry(size * 0.4, 8, 6);
        
        // 创建发光材质
        const material = new THREE.MeshBasicMaterial({
            color: color,
            transparent: true,
            opacity: 0.8
        });

        const particle = new THREE.Mesh(geometry, material);
        particle.position.copy(position);

        // 添加外层光晕 - 减小尺寸
        const glowGeometry = new THREE.SphereGeometry(size * 0.48, 8, 6);
        const glowMaterial = new THREE.MeshBasicMaterial({
            color: color,
            transparent: true,
            opacity: 0.25
        });

        const glow = new THREE.Mesh(glowGeometry, glowMaterial);
        particle.add(glow);

        // 动画属性
        particle.userData = {
            originalPosition: position.clone(),
            originalSize: size,
            originalColor: new THREE.Color(color),
            pulsation: Math.random() * Math.PI * 2,
            glowObject: glow
        };

        this.particles.push(particle);
        this.scene.add(particle);

        return particle;
    }

    /**
     * 更新粒子动画 - 性能优化版本
     */
    updateParticles(time) {
        // 性能优化：仅更新可见粒子
        const deltaTime = time - (this.lastUpdateTime || 0);
        if (deltaTime < 16) return; // 限制为60FPS
        this.lastUpdateTime = time;
        
        // 预计算时间因子
        const timeFactors = {
            pulsation: time * 0.001,
            glow: time * 0.0015
        };
        
        this.particles.forEach(particle => {
            const userData = particle.userData;
            if (!userData) return;
            
            // 减小脉动幅度，增加平滑度
            const pulsation = Math.sin(timeFactors.pulsation + userData.pulsation) * 0.08 + 1;
            particle.scale.setScalar(pulsation);
            
            // 更平滑的光晕效果
            if (userData.glowObject) {
                const glowOpacity = 0.15 + Math.sin(timeFactors.glow + userData.pulsation) * 0.08;
                userData.glowObject.material.opacity = Math.max(0.05, glowOpacity);
            }

            // 轻微的浮动
            const float = Math.sin(time * 0.001 + userData.pulsation) * 0.5;
            particle.position.y = userData.originalPosition.y + float;
        });
    }

    /**
     * 创建连接线
     */
    createConnection(particle1, particle2, opacity = 0.3) {
        const points = [particle1.position.clone(), particle2.position.clone()];
        const geometry = new THREE.BufferGeometry().setFromPoints(points);
        
        const material = new THREE.LineBasicMaterial({
            color: 0x4ecdc4,
            transparent: true,
            opacity: opacity
        });

        const line = new THREE.Line(geometry, material);
        line.userData = {
            particle1: particle1,
            particle2: particle2,
            originalOpacity: opacity
        };

        this.connections.push(line);
        this.scene.add(line);

        return line;
    }

    /**
     * 更新连接线
     */
    updateConnections() {
        this.connections.forEach(connection => {
            const userData = connection.userData;
            
            // 更新线的端点位置
            const positions = connection.geometry.attributes.position.array;
            positions[0] = userData.particle1.position.x;
            positions[1] = userData.particle1.position.y;
            positions[2] = userData.particle1.position.z;
            positions[3] = userData.particle2.position.x;
            positions[4] = userData.particle2.position.y;
            positions[5] = userData.particle2.position.z;
            
            connection.geometry.attributes.position.needsUpdate = true;
        });
    }

    /**
     * 创建搜索涟漪效果 - 最佳实践版本
     */
    createSearchRipple(position, maxRadius = 3.5, duration = 1200) {
        // 外圈涟漪（最佳实践尺寸）
        const outerRippleGeometry = new THREE.RingGeometry(0, 0.8, 32);
        const outerRippleMaterial = new THREE.MeshBasicMaterial({
            color: 0xffd93d,
            transparent: true,
            opacity: 0.5, // 降低初始透明度
            side: THREE.DoubleSide
        });

        const outerRipple = new THREE.Mesh(outerRippleGeometry, outerRippleMaterial);
        outerRipple.position.copy(position);
        outerRipple.lookAt(new THREE.Vector3(0, 0, 0));

        // 内圈涟漪（更亮更小，最佳实践尺寸）
        const innerRippleGeometry = new THREE.RingGeometry(0, 0.5, 32);
        const innerRippleMaterial = new THREE.MeshBasicMaterial({
            color: 0xffffff,
            transparent: true,
            opacity: 0.7, // 降低初始透明度，避免过于刺眼
            side: THREE.DoubleSide
        });

        const innerRipple = new THREE.Mesh(innerRippleGeometry, innerRippleMaterial);
        innerRipple.position.copy(position);
        innerRipple.lookAt(new THREE.Vector3(0, 0, 0));

        const rippleData = {
            startTime: Date.now(),
            duration: duration,
            maxRadius: maxRadius,
            startRadius: 0,
            outerRipple: outerRipple,
            innerRipple: innerRipple
        };

        outerRipple.userData = rippleData;
        innerRipple.userData = rippleData;

        this.searchRipples.push(outerRipple, innerRipple);
        this.scene.add(outerRipple);
        this.scene.add(innerRipple);

        return outerRipple;
    }

    /**
     * 更新搜索涟漪 - 支持双层效果
     */
    updateSearchRipples() {
        const currentTime = Date.now();
        
        this.searchRipples = this.searchRipples.filter(ripple => {
            const userData = ripple.userData;
            const elapsed = currentTime - userData.startTime;
            const progress = elapsed / userData.duration;

            if (progress >= 1) {
                this.scene.remove(ripple);
                ripple.geometry.dispose();
                ripple.material.dispose();
                return false;
            }

            // 缓动函数让扩展更平滑
            const easeProgress = this.easeOutCubic(progress);
            
            // 更新涟漪大小和透明度
            const currentRadius = userData.startRadius + (userData.maxRadius - userData.startRadius) * easeProgress;
            ripple.scale.setScalar(currentRadius);
            
            // 不同层次的透明度变化
            if (ripple.material.color.getHex() === 0xffffff) {
                // 内圈（白色）- 更快淡出
                ripple.material.opacity = 0.9 * (1 - progress * progress);
            } else {
                // 外圈（黄色）- 慢慢淡出
                ripple.material.opacity = 0.6 * (1 - progress);
            }

            return true;
        });
    }
    
    /**
     * 缓出三次方函数
     */
    easeOutCubic(t) {
        return 1 - Math.pow(1 - t, 3);
    }
    
    /**
     * 创建脉冲效果
     */
    createPulseEffect(particle, color = 0xffd93d, duration = 2000) {
        if (!particle || !particle.position) return;
        
        const pulseGeometry = new THREE.SphereGeometry(0.1, 8, 6);
        const pulseMaterial = new THREE.MeshBasicMaterial({
            color: color,
            transparent: true,
            opacity: 0.8
        });
        
        const pulse = new THREE.Mesh(pulseGeometry, pulseMaterial);
        pulse.position.copy(particle.position);
        
        pulse.userData = {
            startTime: Date.now(),
            duration: duration,
            originalColor: new THREE.Color(color)
        };
        
        this.scene.add(pulse);
        
        // 脉冲动画
        let progress = 0;
        const animate = () => {
            progress += 0.04;
            if (progress <= 1) {
                const scale = 1 + Math.sin(progress * Math.PI * 6) * 0.3;
                pulse.scale.setScalar(scale);
                pulse.material.opacity = 0.8 * (1 - progress);
                
                requestAnimationFrame(animate);
            } else {
                this.scene.remove(pulse);
                pulse.geometry.dispose();
                pulse.material.dispose();
            }
        };
        animate();
    }

    /**
     * 高亮粒子 - 增强版本，支持更好的用户体验
     */
    highlightParticle(particle, color = 0xffd93d, intensity = 1.6) {
        // 安全检查
        if (!particle || !particle.userData || !particle.material) {
            console.warn('高亮粒子失败: 无效的粒子对象');
            return;
        }

        // 确保originalColor存在
        if (!particle.userData.originalColor) {
            particle.userData.originalColor = particle.material.color.clone();
        }
        
        const originalColor = particle.userData.originalColor.clone();
        
        // 动画到高亮颜色 - 更流畅的动画
        const startColor = particle.material.color.clone();
        const endColor = new THREE.Color(color);
        const originalSize = particle.userData.originalSize || 1.5;
        
        // 取消之前的动画
        if (particle.userData.highlightAnimation) {
            cancelAnimationFrame(particle.userData.highlightAnimation);
        }
        
        let progress = 0;
        const animate = () => {
            progress += 0.08; // 更平滑的动画速度
            if (progress <= 1) {
                // 使用缓动函数
                const easeProgress = this.easeInOutCubic(progress);
                
                particle.material.color.lerpColors(startColor, endColor, easeProgress);
                particle.scale.setScalar(originalSize * (1 + intensity * 0.15 * easeProgress));
                
                // 添加发光效果
                if (particle.userData.glowObject) {
                    particle.userData.glowObject.material.opacity = 0.6 + 0.4 * easeProgress;
                }
                
                particle.userData.highlightAnimation = requestAnimationFrame(animate);
            } else {
                particle.userData.highlightAnimation = null;
            }
        };
        animate();

        // 创建适中的高亮涟漪（最佳实践尺寸）
        this.createSearchRipple(particle.position.clone(), 3.5, 1200);
        
        // 添加脉冲效果
        this.createPulseEffect(particle, color, 2000);
    }

    /**
     * 重置粒子高亮 - 安全版本
     */
    resetParticleHighlight(particle) {
        // 安全检查
        if (!particle || !particle.userData || !particle.material) {
            return;
        }
        
        // 取消当前的高亮动画
        if (particle.userData.highlightAnimation) {
            cancelAnimationFrame(particle.userData.highlightAnimation);
            particle.userData.highlightAnimation = null;
        }
        
        const originalColor = particle.userData.originalColor || new THREE.Color(0x4ecdc4);
        const originalSize = particle.userData.originalSize || 1.5;
        
        // 动画回到原始颜色 - 更平滑
        const startColor = particle.material.color.clone();
        
        let progress = 0;
        const animate = () => {
            progress += 0.08;
            if (progress <= 1) {
                const easeProgress = this.easeInOutCubic(progress);
                particle.material.color.lerpColors(startColor, originalColor, easeProgress);
                
                // 平滑缩放回原始大小
                const currentScale = particle.scale.x;
                const targetScale = originalSize;
                particle.scale.setScalar(currentScale + (targetScale - currentScale) * easeProgress);
                
                // 重置发光效果
                if (particle.userData.glowObject) {
                    particle.userData.glowObject.material.opacity = 0.3;
                }
                
                requestAnimationFrame(animate);
            }
        };
        animate();
    }

    /**
     * 创建聚类动画
     */
    animateClusterFormation(particles, clusters, duration = 3000) {
        const startTime = Date.now();
        
        const animate = () => {
            const elapsed = Date.now() - startTime;
            const progress = Math.min(elapsed / duration, 1);
            
            // 使用缓动函数
            const easeProgress = this.easeInOutCubic(progress);
            
            particles.forEach((particle, index) => {
                const clusterIndex = this.findParticleCluster(index, clusters);
                if (clusterIndex !== -1) {
                    const cluster = clusters[clusterIndex];
                    const targetColor = new THREE.Color(cluster.color);
                    
                    // 颜色过渡
                    particle.material.color.lerpColors(
                        particle.userData.originalColor, 
                        targetColor, 
                        easeProgress
                    );
                }
            });
            
            if (progress < 1) {
                requestAnimationFrame(animate);
            }
        };
        
        animate();
    }

    /**
     * 缓动函数
     */
    easeInOutCubic(t) {
        return t < 0.5 ? 4 * t * t * t : (t - 1) * (2 * t - 2) * (2 * t - 2) + 1;
    }
    
    /**
     * 缓入函数
     */
    easeInCubic(t) {
        return t * t * t;
    }

    /**
     * 找到粒子所属的聚类
     */
    findParticleCluster(particleIndex, clusters) {
        for (let i = 0; i < clusters.length; i++) {
            if (clusters[i].points && clusters[i].points.includes(particleIndex)) {
                return i;
            }
        }
        return -1;
    }

    /**
     * 更新星空旋转
     */
    updateStarField(time) {
        if (this.starField) {
            this.starField.rotation.y = time * 0.00005;
            this.starField.rotation.z = time * 0.00002;
        }
    }

    /**
     * 切换动画状态
     */
    toggleAnimation() {
        this.isAnimating = !this.isAnimating;
        return this.isAnimating;
    }

    /**
     * 主更新循环
     */
    update(time) {
        if (!this.isAnimating) return;

        this.updateParticles(time);
        this.updateConnections();
        this.updateSearchRipples();
        this.updateStarField(time);
    }

    /**
     * 平滑颜色过渡动画
     */
    animateColorTransition(object, targetColor, duration = 300) {
        if (!object || !object.material) return;
        
        const startColor = object.material.color.clone();
        const startTime = performance.now();
        
        const animate = (currentTime) => {
            const elapsed = currentTime - startTime;
            const progress = Math.min(elapsed / duration, 1);
            
            const eased = this.easeInOutCubic(progress);
            object.material.color.lerpColors(startColor, targetColor, eased);
            
            if (progress < 1) {
                requestAnimationFrame(animate);
            }
        };
        
        requestAnimationFrame(animate);
    }
    
    /**
     * 平滑缩放过渡动画
     */
    animateScaleTransition(object, targetScale, duration = 300) {
        if (!object) return;
        
        const startScale = object.scale.x;
        const startTime = performance.now();
        
        const animate = (currentTime) => {
            const elapsed = currentTime - startTime;
            const progress = Math.min(elapsed / duration, 1);
            
            const eased = this.easeInOutCubic(progress);
            const currentScale = startScale + (targetScale - startScale) * eased;
            object.scale.setScalar(currentScale);
            
            if (progress < 1) {
                requestAnimationFrame(animate);
            }
        };
        
        requestAnimationFrame(animate);
    }
    
    /**
     * 平滑透明度过渡动画
     */
    animateOpacityTransition(object, targetOpacity, duration = 300) {
        if (!object || !object.material) return;
        
        const startOpacity = object.material.opacity;
        const startTime = performance.now();
        
        const animate = (currentTime) => {
            const elapsed = currentTime - startTime;
            const progress = Math.min(elapsed / duration, 1);
            
            const eased = this.easeInOutCubic(progress);
            object.material.opacity = startOpacity + (targetOpacity - startOpacity) * eased;
            
            if (progress < 1) {
                requestAnimationFrame(animate);
            }
        };
        
        requestAnimationFrame(animate);
    }

    /**
     * 清理资源
     */
    dispose() {
        // 清理粒子
        this.particles.forEach(particle => {
            this.scene.remove(particle);
            particle.geometry.dispose();
            particle.material.dispose();
        });
        
        // 清理连接线
        this.connections.forEach(connection => {
            this.scene.remove(connection);
            connection.geometry.dispose();
            connection.material.dispose();
        });
        
        // 清理涟漪
        this.searchRipples.forEach(ripple => {
            this.scene.remove(ripple);
            ripple.geometry.dispose();
            ripple.material.dispose();
        });
        
        // 清理星空
        if (this.starField) {
            this.scene.remove(this.starField);
            this.starField.geometry.dispose();
            this.starField.material.dispose();
        }

        this.particles = [];
        this.connections = [];
        this.searchRipples = [];
    }
}

// 导出到全局范围
window.EffectsEngine = EffectsEngine;