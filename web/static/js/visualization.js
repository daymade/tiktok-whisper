/**
 * 主可视化模块
 * 负责 Three.js 场景管理、交互处理和数据渲染
 */

class EmbeddingVisualizer {
    constructor(container) {
        this.container = container;
        this.scene = null;
        this.camera = null;
        this.renderer = null;
        this.controls = null;
        this.effectsEngine = null;
        this.clusteringEngine = null;
        
        this.embeddings = [];
        this.particles = [];
        this.reducedCoords = [];
        this.currentProvider = 'gemini';
        this.currentMethod = 'umap';
        
        this.raycaster = new THREE.Raycaster();
        this.raycaster.params.Points.threshold = 3; // 减小阈值提高精度
        this.raycaster.far = 100; // 优化检测距离
        
        // 性能优化：鼠标事件节流
        this.lastMouseMoveTime = 0;
        this.mouseThrottleDelay = 16; // ~60fps
        this.nearbyParticles = [];
        this.mouse = new THREE.Vector2();
        this.selectedParticle = null;
        this.hoveredParticle = null;
        this.isUserInteracting = false; // 跟踪用户交互状态
        
        this.animationId = null;
        this.isInitialized = false;

        this.initThreeJS();
        this.setupEventListeners();
    }

    /**
     * 初始化 Three.js 场景
     */
    initThreeJS() {
        console.log('[DEBUG] 开始初始化Three.js...');
        
        // 创建场景
        this.scene = new THREE.Scene();
        this.scene.fog = new THREE.Fog(0x0a0a0a, 50, 200);
        console.log('[DEBUG] 场景创建完成');

        // 创建摄像机
        this.camera = new THREE.PerspectiveCamera(
            75,
            this.container.clientWidth / this.container.clientHeight,
            0.1,
            1000
        );
        this.camera.position.set(30, 20, 30);
        this.camera.lookAt(0, 0, 0);
        console.log('[DEBUG] 摄像机创建完成');

        // 创建渲染器
        this.renderer = new THREE.WebGLRenderer({ 
            canvas: this.container.querySelector('#three-canvas'),
            antialias: true,
            alpha: true
        });
        this.renderer.setSize(this.container.clientWidth, this.container.clientHeight);
        this.renderer.setPixelRatio(Math.min(window.devicePixelRatio, 2));
        console.log('[DEBUG] 渲染器创建完成');

        // 创建控制器
        console.log('[DEBUG] 开始创建OrbitControls，THREE.OrbitControls类型:', typeof THREE.OrbitControls);
        this.controls = new THREE.OrbitControls(this.camera, this.renderer.domElement);
        console.log('[DEBUG] OrbitControls创建完成，实例类型:', typeof this.controls);
        
        // 临时修复：如果OrbitControls没有addEventListener方法，添加一个
        if (!this.controls.addEventListener) {
            console.warn('[DEBUG] OrbitControls缺少addEventListener方法，添加polyfill');
            this.controls.addEventListener = (type, listener, options) => {
                console.log('[DEBUG] 检测到addEventListener调用:', { type, listener: typeof listener, options });
                console.trace('[DEBUG] addEventListener调用栈:');
                // 这里我们可以选择忽略或实现适当的行为
                // 对于OrbitControls，通常change事件可以通过controls.update()在animation loop中处理
            };
        }
        this.controls.enableDamping = true;
        this.controls.dampingFactor = 0.05;
        this.controls.enableZoom = true;
        this.controls.enablePan = true;
        this.controls.maxDistance = 100;
        this.controls.minDistance = 5;
        console.log('[DEBUG] OrbitControls属性设置完成');
        
        // 监听控制器事件来跟踪用户交互状态（备用方案）
        // 由于我们的OrbitControls版本可能不支持addEventListener，使用鼠标事件检测
        console.log('[DEBUG] 开始设置鼠标事件监听器');
        let isDragging = false;
        this.renderer.domElement.addEventListener('mousedown', () => {
            isDragging = true;
            this.isUserInteracting = true;
            this.hideQuickInfo();
        });
        
        this.renderer.domElement.addEventListener('mouseup', () => {
            if (isDragging) {
                setTimeout(() => {
                    this.isUserInteracting = false;
                }, 10); // 减少延迟时间避免阻塞点击事件
            }
            isDragging = false;
        });
        
        this.renderer.domElement.addEventListener('mouseleave', () => {
            isDragging = false;
            this.isUserInteracting = false;
        });

        // 初始化效果引擎
        this.effectsEngine = new EffectsEngine(this.scene, this.renderer);
        
        // 初始化聚类引擎
        this.clusteringEngine = new ClusteringEngine();

        // 添加环境光
        const ambientLight = new THREE.AmbientLight(0x404040, 0.6);
        this.scene.add(ambientLight);

        // 添加方向光
        const directionalLight = new THREE.DirectionalLight(0xffffff, 0.8);
        directionalLight.position.set(50, 50, 50);
        directionalLight.castShadow = true;
        this.scene.add(directionalLight);

        console.log('[DEBUG] initThreeJS方法完成，所有组件初始化成功');
        this.isInitialized = true;
        this.startRenderLoop();
        console.log('[DEBUG] 渲染循环启动');
    }

    /**
     * 设置事件监听器 - 性能优化版本
     */
    setupEventListeners() {
        console.log('[DEBUG] 开始设置事件监听器...');
        
        // 鼠标事件 - 使用节流优化性能
        this.renderer.domElement.addEventListener('mousemove', (event) => {
            const now = performance.now();
            if (now - this.lastMouseMoveTime > this.mouseThrottleDelay) {
                this.onMouseMove(event);
                this.lastMouseMoveTime = now;
            }
        });
        
        this.renderer.domElement.addEventListener('click', (event) => {
            this.onMouseClick(event);
        });
        
        this.renderer.domElement.addEventListener('mouseleave', () => {
            this.hideQuickInfo();
            if (this.hoveredParticle && this.hoveredParticle !== this.selectedParticle) {
                this.resetHoverEffect(this.hoveredParticle);
                this.hoveredParticle = null;
            }
            this.renderer.domElement.style.cursor = 'default';
        });
        
        // 窗口大小调整
        window.addEventListener('resize', this.onWindowResize.bind(this));
        
        // 键盘事件
        document.addEventListener('keydown', this.onKeyDown.bind(this));
        console.log('[DEBUG] 事件监听器设置完成 - 使用箭头函数绑定');
    }

    /**
     * 加载并可视化嵌入数据
     */
    async loadEmbeddings(provider = 'gemini', limit = 100) {
        try {
            console.log(`正在加载 ${provider} embeddings...`);
            
            const response = await fetch(`/api/embeddings?provider=${provider}&limit=${limit}`);
            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }
            
            this.embeddings = await response.json();
            console.log(`成功加载 ${this.embeddings.length} 个 embeddings`);
            
            if (this.embeddings.length === 0) {
                throw new Error('没有找到嵌入数据');
            }

            this.currentProvider = provider;
            await this.visualizeEmbeddings();
            
            return this.embeddings;
        } catch (error) {
            console.error('加载嵌入数据失败:', error);
            throw error;
        }
    }

    /**
     * 可视化嵌入数据
     */
    async visualizeEmbeddings(method = 'umap') {
        if (this.embeddings.length === 0) {
            console.warn('没有可视化的数据');
            return;
        }

        console.log(`开始可视化 ${this.embeddings.length} 个数据点...`);

        // 清理之前的粒子
        this.clearParticles();

        try {
            // 执行降维
            this.reducedCoords = await this.clusteringEngine.reduceDimensions(
                this.embeddings, 
                method, 
                3
            );

            // 执行增强聚类 - 使用肘部法自动确定聚类数
            const clusters = this.clusteringEngine.performKMeansClustering(this.reducedCoords);
            console.log(`生成了${clusters.length}个聚类，分别包含: ${clusters.map(c => c.points.length).join(', ')}个数据点`);
            
            // 增强聚类分离
            this.reducedCoords = this.clusteringEngine.enhanceClusterSeparation(this.reducedCoords, clusters);

            // 创建粒子
            this.createParticles();

            // 应用聚类颜色
            this.applyClusterColors(clusters);

            // 创建聚类间的连接
            this.createClusterConnections(clusters);

            // 动画效果
            this.effectsEngine.animateClusterFormation(this.particles, clusters);

            console.log('可视化完成');
            this.currentMethod = method;

        } catch (error) {
            console.error('可视化过程出错:', error);
            // 创建基础粒子作为备用
            this.createBasicParticles();
        }
    }

    /**
     * 创建粒子
     */
    createParticles() {
        this.particles = [];
        
        this.reducedCoords.forEach((coord, index) => {
            const position = new THREE.Vector3(coord[0], coord[1], coord[2] || 0);
            const color = 0x4ecdc4; // 默认颜色
            const size = 1.5;
            
            const particle = this.effectsEngine.createParticle(position, color, size);
            particle.userData.embeddingIndex = index;
            particle.userData.embeddingData = this.embeddings[index];
            
            this.particles.push(particle);
        });

        console.log(`创建了 ${this.particles.length} 个粒子`);
    }

    /**
     * 创建基础粒子（备用方案）
     */
    createBasicParticles() {
        this.particles = [];
        
        this.embeddings.forEach((embedding, index) => {
            // 随机分布
            const position = new THREE.Vector3(
                (Math.random() - 0.5) * 40,
                (Math.random() - 0.5) * 40,
                (Math.random() - 0.5) * 40
            );
            
            const particle = this.effectsEngine.createParticle(position, 0x4ecdc4, 1.5);
            particle.userData.embeddingIndex = index;
            particle.userData.embeddingData = embedding;
            
            this.particles.push(particle);
        });

        console.log(`创建了 ${this.particles.length} 个基础粒子`);
    }

    /**
     * 获得感知统一的聚类颜色
     */
    getPerceptuallyUniformColor(index, clusterSize = 1) {
        // 使用黄金比例获得更好的颜色分布
        const goldenRatio = 0.618033988749895;
        const hue = (index * goldenRatio * 360) % 360;
        
        // 根据聚类大小调整饱和度和亮度
        const saturation = 65 + (clusterSize / this.embeddings.length) * 25; // 65-90%
        const lightness = 50 + (index % 4) * 8; // 50-74%
        
        return `hsl(${hue}, ${saturation}%, ${lightness}%)`;
    }
    
    /**
     * 应用聚类颜色 - 增强版本使用感知统一颜色和大小区分
     */
    applyClusterColors(clusters) {
        console.log(`应用${clusters.length}个聚类的增强配色方案...`);
        
        // 按聚类大小排序，大聚类使用更显眼的颜色
        const sortedClusters = clusters.sort((a, b) => b.points.length - a.points.length);
        
        sortedClusters.forEach((cluster, clusterIndex) => {
            const colorHsl = this.getPerceptuallyUniformColor(clusterIndex, cluster.points.length);
            const color = new THREE.Color(colorHsl);
            cluster.color = colorHsl; // 更新cluster对象的颜色
            
            console.log(`聚类 ${clusterIndex + 1}: ${cluster.points.length}个点, 颜色: ${colorHsl}`);
            
            cluster.points.forEach(pointIndex => {
                if (pointIndex < this.particles.length) {
                    const particle = this.particles[pointIndex];
                    
                    // 根据聚类密度调整粒子大小
                    const density = cluster.points.length / this.embeddings.length;
                    const sizeMultiplier = 1 + density * 0.8; // 大聚类粒子更大
                    particle.userData.originalSize = 1.5 * sizeMultiplier;
                    particle.scale.setScalar(particle.userData.originalSize);
                    
                    // 应用颜色
                    particle.material.color.copy(color);
                    particle.userData.originalColor = color.clone();
                    particle.userData.clusterId = cluster.id;
                    particle.userData.clusterSize = cluster.points.length;
                    particle.userData.userInfo = particle.userData.embeddingData?.user || '未知用户';
                    
                    // 增强发光效果 - 大聚类更亮
                    if (particle.userData.glowObject) {
                        particle.userData.glowObject.material.color.copy(color);
                        particle.userData.glowObject.material.opacity = 0.3 + density * 0.4;
                    }
                }
            });
        });
        
        console.log(`✨ 成功应用了 ${clusters.length} 个聚类的增强配色和大小区分`);
    }

    /**
     * 创建聚类间的连接
     */
    createClusterConnections(clusters) {
        // 清理旧连接
        this.effectsEngine.connections.forEach(connection => {
            this.scene.remove(connection);
            connection.geometry.dispose();
            connection.material.dispose();
        });
        this.effectsEngine.connections = [];

        // 在同一聚类内创建少量连接
        clusters.forEach(cluster => {
            if (cluster.points.length > 1) {
                const maxConnections = Math.min(5, cluster.points.length - 1);
                for (let i = 0; i < maxConnections; i++) {
                    const point1Index = cluster.points[i];
                    const point2Index = cluster.points[(i + 1) % cluster.points.length];
                    
                    if (point1Index < this.particles.length && point2Index < this.particles.length) {
                        this.effectsEngine.createConnection(
                            this.particles[point1Index],
                            this.particles[point2Index],
                            0.1
                        );
                    }
                }
            }
        });
    }

    /**
     * 搜索相似嵌入
     */
    async searchSimilar(query, provider = null, limit = 10) {
        try {
            const searchProvider = provider || this.currentProvider;
            console.log(`搜索相似内容: "${query}"`);

            const response = await fetch(
                `/api/embeddings/search?q=${encodeURIComponent(query)}&provider=${searchProvider}&limit=${limit}`
            );
            
            if (!response.ok) {
                throw new Error(`搜索失败: ${response.status}`);
            }

            const results = await response.json();
            console.log(`找到 ${results.length} 个相似结果`);

            // 高亮搜索结果
            this.highlightSearchResults(results);

            return results;
        } catch (error) {
            console.error('搜索失败:', error);
            throw error;
        }
    }

    /**
     * 高亮搜索结果
     */
    highlightSearchResults(results) {
        // 重置所有粒子
        this.particles.forEach(particle => {
            this.effectsEngine.resetParticleHighlight(particle);
        });

        // 高亮搜索结果
        results.forEach(result => {
            const particle = this.particles.find(p => 
                p.userData.embeddingData && p.userData.embeddingData.id === result.id
            );
            
            if (particle) {
                // 使用最佳实践的搜索高亮强度
                this.effectsEngine.highlightParticle(particle, 0xffd93d, 1.3);
                
                // 创建适中的搜索涟漪（最佳实践尺寸）
                this.effectsEngine.createSearchRipple(particle.position, 4.5, 1500);
            }
        });
    }

    /**
     * 鼠标移动事件 - 高性能版本优化交互体验
     */
    onMouseMove(event) {
        if (this.isUserInteracting) return; // 避免在拖拽时触发悬停
        
        const rect = this.renderer.domElement.getBoundingClientRect();
        this.mouse.x = ((event.clientX - rect.left) / rect.width) * 2 - 1;
        this.mouse.y = -((event.clientY - rect.top) / rect.height) * 2 + 1;

        this.raycaster.setFromCamera(this.mouse, this.camera);
        
        // 性能优化：空间预过滤 + 减少检测范围
        const intersects = this.raycaster.intersectObjects(this.particles, false);
        const nearbyParticles = intersects.filter(intersect => intersect.distance < 50);

        // 处理悬停效果
        if (nearbyParticles.length > 0) {
            const newHovered = nearbyParticles[0].object;
            
            if (this.hoveredParticle !== newHovered) {
                // 重置之前悬停的粒子
                if (this.hoveredParticle && this.hoveredParticle !== this.selectedParticle) {
                    this.resetHoverEffect(this.hoveredParticle);
                }
                
                // 设置新的悬停粒子
                if (newHovered !== this.selectedParticle) {
                    this.hoveredParticle = newHovered;
                    this.applyHoverEffect(this.hoveredParticle);
                    
                    // 更改鼠标样式
                    this.renderer.domElement.style.cursor = 'pointer';
                    
                    // 显示快捷信息
                    this.showQuickInfo(newHovered, event);
                }
            }
        } else {
            if (this.hoveredParticle && this.hoveredParticle !== this.selectedParticle) {
                this.resetHoverEffect(this.hoveredParticle);
                this.hoveredParticle = null;
                this.renderer.domElement.style.cursor = 'default';
                this.hideQuickInfo();
            }
        }
    }

    /**
     * 鼠标点击事件 - 完全重构版本
     */
    onMouseClick(event) {
        console.log('[DEBUG] 点击事件触发:', { 
            isUserInteracting: this.isUserInteracting,
            particleCount: this.particles.length
        });
        
        this.raycaster.setFromCamera(this.mouse, this.camera);
        
        // 使用递归检测以捕获子对象（如glow effects）
        const intersects = this.raycaster.intersectObjects(this.particles, true);
        console.log('[DEBUG] 射线检测结果:', intersects.length, '个相交对象');

        if (intersects.length > 0) {
            const clickedObject = intersects[0].object;
            console.log('[DEBUG] 原始点击对象:', {
                type: clickedObject.type,
                hasEmbeddingData: !!clickedObject.userData?.embeddingData,
                isParticle: this.particles.includes(clickedObject)
            });
            
            // 多重策略查找真正的粒子对象
            const actualParticle = this.findActualParticle(clickedObject);
            
            if (actualParticle && actualParticle.userData?.embeddingData) {
                console.log('[DEBUG] 找到有效粒子:', {
                    id: actualParticle.userData.embeddingData.id,
                    user: actualParticle.userData.embeddingData.user
                });
                
                this.selectParticle(actualParticle);
                this.addClickFeedback(actualParticle.position);
            } else {
                console.log('[DEBUG] 未找到有效粒子，尝试位置匹配');
                const nearestParticle = this.findNearestParticleByPosition(intersects[0].point);
                if (nearestParticle) {
                    console.log('[DEBUG] 通过位置找到粒子:', nearestParticle.userData.embeddingData?.id);
                    this.selectParticle(nearestParticle);
                    this.addClickFeedback(nearestParticle.position);
                } else {
                    console.log('[DEBUG] 完全找不到有效粒子');
                    this.deselectParticle();
                }
            }
        } else {
            console.log('[DEBUG] 没有检测到任何对象，取消选择');
            this.deselectParticle();
        }
    }

    /**
     * 选择粒子 - 防御性编程版本
     */
    selectParticle(particle) {
        // 输入验证
        if (!particle) {
            console.error('[ERROR] selectParticle: 粒子对象为空');
            return;
        }
        
        if (!particle.userData || !particle.userData.embeddingData) {
            console.error('[ERROR] selectParticle: 粒子缺少embeddingData', {
                hasUserData: !!particle.userData,
                userData: particle.userData
            });
            return;
        }
        
        console.log('[DEBUG] 选择粒子:', {
            id: particle.userData.embeddingData.id,
            user: particle.userData.embeddingData.user,
            hasPosition: !!particle.position
        });

        // 重置之前选择的粒子
        if (this.selectedParticle) {
            this.effectsEngine.resetParticleHighlight(this.selectedParticle);
        }

        this.selectedParticle = particle;
        
        // 优化的选中效果（红色，最佳实践大小）
        this.effectsEngine.highlightParticle(particle, 0xff4757, 1.3);

        // 显示信息面板
        this.showInfoPanel(particle.userData.embeddingData);

        // 创建适中的选择涟漪（最佳实践尺寸）
        this.effectsEngine.createSearchRipple(particle.position.clone(), 3.5, 1200);
        
        // 高亮相同聚类的其他粒子
        this.highlightSameCluster(particle);
        
        console.log(`✅ 成功选中粒子 ID: ${particle.userData.embeddingData.id}, 用户: ${particle.userData.embeddingData.user}`);
    }
    
    /**
     * 查找真正的粒子对象 - 多重策略
     */
    findActualParticle(object) {
        // 策略1: 如果对象本身就在particles数组中
        if (this.particles.includes(object)) {
            return object;
        }
        
        // 策略2: 向上遍历父对象
        let current = object;
        while (current) {
            if (this.particles.includes(current)) {
                return current;
            }
            // 检查当前对象是否有embeddingData
            if (current.userData?.embeddingData) {
                // 验证这个对象是否在particles数组中
                const found = this.particles.find(p => p === current);
                if (found) return found;
            }
            current = current.parent;
        }
        
        // 策略3: 通过场景遍历查找
        if (object.parent) {
            const siblings = object.parent.children;
            for (let sibling of siblings) {
                if (this.particles.includes(sibling) && sibling.userData?.embeddingData) {
                    return sibling;
                }
            }
        }
        
        return null;
    }
    
    /**
     * 根据位置查找最近的粒子
     */
    findNearestParticleByPosition(position) {
        let nearestParticle = null;
        let minDistance = Infinity;
        
        for (let particle of this.particles) {
            if (!particle.userData?.embeddingData) continue;
            
            const distance = particle.position.distanceTo(position);
            if (distance < minDistance) {
                minDistance = distance;
                nearestParticle = particle;
            }
        }
        
        // 只返回距离合理的粒子（小于5单位）
        return minDistance < 5 ? nearestParticle : null;
    }

    /**
     * 取消选择粒子
     */
    deselectParticle() {
        if (this.selectedParticle) {
            this.effectsEngine.resetParticleHighlight(this.selectedParticle);
            this.selectedParticle = null;
        }
        
        // 重置所有聚类高亮
        this.resetClusterHighlight();
        
        this.hideInfoPanel();
        this.hideQuickInfo();
    }
    
    /**
     * 应用悬停效果 - 最佳实践版本
     */
    applyHoverEffect(particle) {
        if (!particle || !particle.userData) return;
        
        const originalSize = particle.userData.originalSize || 1.2;
        const targetScale = originalSize * 1.15; // 行业最佳实践：15%增加
        
        // 平滑缩放动画（最优时间）
        this.animateParticleScale(particle, targetScale, 150);
        
        // 精细的发光效果
        if (particle.userData.glowObject) {
            this.animateGlowOpacity(particle.userData.glowObject, 0.4, 150);
        }
        
        // 添加微妙的颜色增强
        this.enhanceParticleColor(particle, 0.1);
    }
    
    /**
     * 重置悬停效果 - 平滑动画版本
     */
    resetHoverEffect(particle) {
        if (!particle || !particle.userData) return;
        
        const originalSize = particle.userData.originalSize || 1.2;
        
        // 平滑重置缩放
        this.animateParticleScale(particle, originalSize, 200);
        
        // 平滑重置发光效果
        if (particle.userData.glowObject) {
            this.animateGlowOpacity(particle.userData.glowObject, 0.25, 200);
        }
        
        // 重置颜色增强
        this.resetParticleColor(particle);
    }
    
    /**
     * 显示快捷信息
     */
    showQuickInfo(particle, event) {
        const data = particle.userData.embeddingData;
        if (!data || !event) return;
        
        // 创建或更新tooltip
        let tooltip = document.getElementById('particle-tooltip');
        if (!tooltip) {
            tooltip = document.createElement('div');
            tooltip.id = 'particle-tooltip';
            tooltip.style.cssText = `
                position: fixed;
                background: rgba(0, 0, 0, 0.9);
                color: white;
                padding: 8px 12px;
                border-radius: 8px;
                font-size: 12px;
                pointer-events: none;
                z-index: 1000;
                max-width: 280px;
                box-shadow: 0 4px 20px rgba(0, 0, 0, 0.5);
                border: 1px solid #4ecdc4;
                backdrop-filter: blur(10px);
                transition: opacity 0.2s ease;
            `;
            document.body.appendChild(tooltip);
        }
        
        const preview = data.textPreview || data.text || '无内容';
        const clusterId = particle.userData.clusterId || '未分类';
        const clusterSize = particle.userData.clusterSize || 0;
        const userInfo = particle.userData.userInfo || data.user || '未知用户';
        const dimensions = data.embedding ? data.embedding.length : '未知';
        
        // 聚类密度信息
        const density = clusterSize / this.embeddings.length;
        const densityDesc = density > 0.3 ? '大聚类' : density > 0.1 ? '中聚类' : '小聚类';
        
        tooltip.innerHTML = `
            <div style="color: #4ecdc4; font-weight: bold; margin-bottom: 4px;">✨ ID: ${data.id}</div>
            <div style="color: #ffd93d; font-size: 11px; margin-bottom: 2px;">👤 ${userInfo}</div>
            <div style="color: #ff6b6b; font-size: 10px; margin-bottom: 2px;">🎯 聚类 ${clusterId} (${clusterSize}个点 - ${densityDesc})</div>
            <div style="color: #98d8c8; font-size: 10px; margin-bottom: 4px;">📊 维度: ${dimensions} | 🔍 密度: ${(density * 100).toFixed(1)}%</div>
            <div style="color: #e0e0e0; font-size: 11px; line-height: 1.3;">${preview.substring(0, 80)}${preview.length > 80 ? '...' : ''}</div>
        `;
        
        // 智能定位 - 避免超出屏幕
        const mouseX = event.clientX;
        const mouseY = event.clientY;
        const tooltipWidth = 280;
        const tooltipHeight = 100; // 估计高度
        
        let left = mouseX + 15;
        let top = mouseY - tooltipHeight - 15;
        
        // 边界检测
        if (left + tooltipWidth > window.innerWidth) {
            left = mouseX - tooltipWidth - 15;
        }
        if (top < 0) {
            top = mouseY + 15;
        }
        
        tooltip.style.left = left + 'px';
        tooltip.style.top = top + 'px';
        tooltip.style.display = 'block';
        tooltip.style.opacity = '1';
    }
    
    /**
     * 隐藏tooltip
     */
    hideQuickInfo() {
        const tooltip = document.getElementById('particle-tooltip');
        if (tooltip) {
            tooltip.style.display = 'none';
        }
    }
    
    /**
     * 添加点击反馈 - 最佳实践版本
     */
    addClickFeedback(position) {
        // 创建点击波纹（适中尺寸）
        const clickGeometry = new THREE.RingGeometry(0, 1.5, 16);
        const clickMaterial = new THREE.MeshBasicMaterial({
            color: 0xffffff,
            transparent: true,
            opacity: 0.6,
            side: THREE.DoubleSide
        });
        
        const clickEffect = new THREE.Mesh(clickGeometry, clickMaterial);
        clickEffect.position.copy(position);
        clickEffect.lookAt(this.camera.position);
        
        this.scene.add(clickEffect);
        
        // 优化的动画（最大放大2倍，符合最佳实践）
        let progress = 0;
        const animate = () => {
            progress += 0.08; // 更平滑的动画速度
            if (progress <= 1) {
                // 使用easing函数和最佳实践缩放比例
                const eased = this.easeOutCubic(progress);
                clickEffect.scale.setScalar(1 + eased * 1.2); // 最大放大2.2倍
                clickEffect.material.opacity = 0.6 * (1 - progress * progress); // 平方衰减
                requestAnimationFrame(animate);
            } else {
                this.scene.remove(clickEffect);
                clickEffect.geometry.dispose();
                clickEffect.material.dispose();
            }
        };
        animate();
    }
    
    /**
     * 高亮相同聚类
     */
    highlightSameCluster(particle) {
        if (!particle.userData.clusterId) return;
        
        const clusterId = particle.userData.clusterId;
        const clusterColor = particle.userData.originalColor;
        
        this.particles.forEach(p => {
            if (p !== particle && p.userData.clusterId === clusterId) {
                // 轻微高亮相同聚类的粒子（最佳实践缩放）
                p.scale.setScalar((p.userData.originalSize || 1.5) * 1.08); // 减小到 8%
                if (p.userData.glowObject) {
                    p.userData.glowObject.material.opacity = 0.35; // 轻微减小发光
                }
            }
        });
    }
    
    /**
     * 重置聚类高亮
     */
    resetClusterHighlight() {
        this.particles.forEach(p => {
            if (p !== this.selectedParticle && p !== this.hoveredParticle) {
                p.scale.setScalar(p.userData.originalSize || 1.5);
                if (p.userData.glowObject) {
                    p.userData.glowObject.material.opacity = 0.3;
                }
            }
        });
    }

    /**
     * 显示信息面板 - 增强错误处理
     */
    showInfoPanel(data) {
        // 防御性检查
        if (!data) {
            console.error('[ERROR] showInfoPanel: 数据为空');
            return;
        }
        
        const infoPanel = document.getElementById('info-panel');
        if (!infoPanel) {
            console.warn('[WARN] showInfoPanel: 找不到info-panel元素');
            return;
        }

        // 安全设置元素内容
        const setElementText = (id, value) => {
            const element = document.getElementById(id);
            if (element) {
                element.textContent = value || '-';
            } else {
                console.warn(`[WARN] 找不到元素: ${id}`);
            }
        };

        setElementText('info-id', data.id);
        setElementText('info-user', data.user);
        setElementText('info-text', data.text || data.textPreview);
        setElementText('info-dimensions', data.embedding ? data.embedding.length : null);
        setElementText('info-created', data.createdAt ? new Date(data.createdAt).toLocaleDateString('zh-CN') : null);

        infoPanel.classList.remove('hidden');
        console.log('[DEBUG] 信息面板已显示:', data.id);
    }

    /**
     * 隐藏信息面板
     */
    hideInfoPanel() {
        const infoPanel = document.getElementById('info-panel');
        if (infoPanel) {
            infoPanel.classList.add('hidden');
        }
    }

    /**
     * 键盘事件
     */
    onKeyDown(event) {
        switch (event.code) {
            case 'Escape':
                this.deselectParticle();
                break;
            case 'Space':
                event.preventDefault();
                this.effectsEngine.toggleAnimation();
                break;
            case 'KeyR':
                this.resetView();
                break;
        }
    }

    /**
     * 窗口大小调整
     */
    onWindowResize() {
        if (!this.isInitialized) return;

        this.camera.aspect = this.container.clientWidth / this.container.clientHeight;
        this.camera.updateProjectionMatrix();
        this.renderer.setSize(this.container.clientWidth, this.container.clientHeight);
    }

    /**
     * 重置视角
     */
    resetView() {
        this.camera.position.set(30, 20, 30);
        this.camera.lookAt(0, 0, 0);
        this.controls.reset();
    }

    /**
     * 更新点大小
     */
    updatePointSize(size) {
        this.particles.forEach(particle => {
            particle.userData.originalSize = size;
            if (particle !== this.selectedParticle && particle !== this.hoveredParticle) {
                particle.scale.setScalar(size);
            }
        });
    }

    /**
     * 清理粒子
     */
    clearParticles() {
        this.particles.forEach(particle => {
            this.scene.remove(particle);
            particle.geometry.dispose();
            particle.material.dispose();
        });
        this.particles = [];
    }

    /**
     * 开始渲染循环 - 性能优化版本
     */
    startRenderLoop() {
        const animate = (time) => {
            this.animationId = requestAnimationFrame(animate);
            
            // 性能监控
            this.updatePerformanceMetrics();
            
            // 更新控制器
            this.controls.update();
            
            // 更新效果
            this.effectsEngine.update(time);
            
            // 渲染场景
            this.renderer.render(this.scene, this.camera);
        };
        
        animate();
    }

    /**
     * 停止渲染循环
     */
    stopRenderLoop() {
        if (this.animationId) {
            cancelAnimationFrame(this.animationId);
            this.animationId = null;
        }
    }

    /**
     * 平滑粒子缩放动画
     */
    animateParticleScale(particle, targetScale, duration = 200) {
        if (!particle || !particle.userData) return;
        
        const startScale = particle.scale.x;
        const startTime = performance.now();
        
        const animate = (currentTime) => {
            const elapsed = currentTime - startTime;
            const progress = Math.min(elapsed / duration, 1);
            
            // 使用cubic-bezier缓动函数
            const eased = this.easeInOutCubic(progress);
            const currentScale = startScale + (targetScale - startScale) * eased;
            
            particle.scale.setScalar(currentScale);
            
            if (progress < 1) {
                requestAnimationFrame(animate);
            }
        };
        
        requestAnimationFrame(animate);
    }
    
    /**
     * 平滑发光透明度动画
     */
    animateGlowOpacity(glowObject, targetOpacity, duration = 200) {
        if (!glowObject || !glowObject.material) return;
        
        const startOpacity = glowObject.material.opacity;
        const startTime = performance.now();
        
        const animate = (currentTime) => {
            const elapsed = currentTime - startTime;
            const progress = Math.min(elapsed / duration, 1);
            
            const eased = this.easeInOutCubic(progress);
            const currentOpacity = startOpacity + (targetOpacity - startOpacity) * eased;
            
            glowObject.material.opacity = currentOpacity;
            
            if (progress < 1) {
                requestAnimationFrame(animate);
            }
        };
        
        requestAnimationFrame(animate);
    }
    
    /**
     * 增强粒子颜色亮度
     */
    enhanceParticleColor(particle, intensity = 0.1) {
        if (!particle || !particle.material || !particle.userData.originalColor) return;
        
        const originalColor = particle.userData.originalColor;
        const enhancedColor = originalColor.clone();
        
        // 增加亮度但保持色相
        enhancedColor.multiplyScalar(1 + intensity);
        particle.material.color.copy(enhancedColor);
    }
    
    /**
     * 重置粒子颜色
     */
    resetParticleColor(particle) {
        if (!particle || !particle.material || !particle.userData.originalColor) return;
        
        particle.material.color.copy(particle.userData.originalColor);
    }
    
    /**
     * 三次贝塞尔缓动函数
     */
    easeInOutCubic(t) {
        return t < 0.5 ? 4 * t * t * t : 1 - Math.pow(-2 * t + 2, 3) / 2;
    }
    
    /**
     * 三次缓出函数（点击反馈优化）
     */
    easeOutCubic(t) {
        return 1 - Math.pow(1 - t, 3);
    }
    
    /**
     * 性能监控
     */
    updatePerformanceMetrics() {
        const now = performance.now();
        if (!this.lastFrameTime) {
            this.lastFrameTime = now;
            return;
        }
        
        const delta = now - this.lastFrameTime;
        this.fps = Math.round(1000 / delta);
        this.lastFrameTime = now;
        
        // 自适应质量调整
        if (this.fps < 45 && this.particles.length > 100) {
            console.log('[PERFORMANCE] 检测到性能下降，启用自适应优化');
            this.mouseThrottleDelay = Math.min(33, this.mouseThrottleDelay + 4); // 降低鼠标检测频率
        } else if (this.fps > 55) {
            this.mouseThrottleDelay = Math.max(16, this.mouseThrottleDelay - 1); // 恢复鼠标检测频率
        }
    }

    /**
     * 销毁可视化器
     */
    dispose() {
        this.stopRenderLoop();
        
        if (this.effectsEngine) {
            this.effectsEngine.dispose();
        }
        
        this.clearParticles();
        
        if (this.renderer) {
            this.renderer.dispose();
        }
        
        if (this.controls) {
            this.controls.dispose();
        }

        // 移除事件监听器
        window.removeEventListener('resize', this.onWindowResize.bind(this));
        document.removeEventListener('keydown', this.onKeyDown.bind(this));
    }
}

// 导出到全局范围
window.EmbeddingVisualizer = EmbeddingVisualizer;