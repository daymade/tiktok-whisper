/**
 * 主应用文件
 * 负责应用初始化、UI 控制和数据管理
 */

class EmbeddingApp {
    constructor() {
        this.visualizer = null;
        this.isLoading = false;
        this.stats = null;
        
        this.initializeApp();
    }

    /**
     * 初始化应用
     */
    async initializeApp() {
        try {
            this.showLoading('正在初始化应用...');
            
            // 初始化可视化器
            const container = document.getElementById('visualization-container');
            this.visualizer = new EmbeddingVisualizer(container);
            
            // 设置 UI 事件监听器
            this.setupUIEventListeners();
            
            // 加载系统统计
            await this.loadStats();
            
            // 加载默认数据
            await this.loadDefaultData();
            
            this.hideLoading();
            
            console.log('应用初始化完成');
        } catch (error) {
            console.error('应用初始化失败:', error);
            this.showError('应用初始化失败: ' + error.message);
        }
    }

    /**
     * 设置 UI 事件监听器
     */
    setupUIEventListeners() {
        // Provider 选择
        const providerSelect = document.getElementById('provider');
        if (providerSelect) {
            providerSelect.addEventListener('change', (e) => {
                this.changeProvider(e.target.value);
            });
        }

        // 聚类方法选择
        const clusterMethodSelect = document.getElementById('cluster-method');
        if (clusterMethodSelect) {
            clusterMethodSelect.addEventListener('change', (e) => {
                this.changeClusterMethod(e.target.value);
            });
        }

        // 搜索功能 - 增强版本
        const searchInput = document.getElementById('search-input');
        const searchBtn = document.getElementById('search-btn');
        
        if (searchInput && searchBtn) {
            // 防抖动搜索
            let searchTimeout;
            const debouncedSearch = (query) => {
                clearTimeout(searchTimeout);
                searchTimeout = setTimeout(() => {
                    this.performSearch(query);
                }, 300); // 300ms 防抖
            };

            // 搜索按钮点击
            searchBtn.addEventListener('click', () => {
                clearTimeout(searchTimeout); // 立即执行搜索
                this.performSearch(searchInput.value);
            });
            
            // Enter 键搜索
            searchInput.addEventListener('keypress', (e) => {
                if (e.key === 'Enter') {
                    clearTimeout(searchTimeout); // 立即执行搜索
                    this.performSearch(searchInput.value);
                }
            });

            // 实时搜索 (可选，用户输入时自动搜索)
            /*
            searchInput.addEventListener('input', (e) => {
                const query = e.target.value.trim();
                if (query.length >= 2) { // 至少2个字符才开始搜索
                    debouncedSearch(query);
                } else if (query.length === 0) {
                    // 清空搜索时重置高亮
                    this.hideSearchResults();
                    if (this.visualizer) {
                        this.visualizer.resetAllHighlights();
                    }
                }
            });
            */

            // 输入验证和提示
            searchInput.addEventListener('input', (e) => {
                const query = e.target.value;
                if (query.length > 200) {
                    e.target.value = query.substring(0, 200);
                    this.showTemporaryMessage('搜索查询不能超过200个字符', 2000);
                }
            });
        }

        // 点大小控制
        const pointSizeSlider = document.getElementById('point-size');
        const pointSizeValue = document.getElementById('point-size-value');
        
        if (pointSizeSlider && pointSizeValue) {
            pointSizeSlider.addEventListener('input', (e) => {
                const size = parseFloat(e.target.value);
                pointSizeValue.textContent = size.toFixed(1);
                if (this.visualizer) {
                    this.visualizer.updatePointSize(size);
                }
            });
        }
        
        // 敏感度控制
        this.setupSensitivityControls();

        // 重置视角按钮
        const resetViewBtn = document.getElementById('reset-view');
        if (resetViewBtn) {
            resetViewBtn.addEventListener('click', () => {
                if (this.visualizer) {
                    this.visualizer.resetView();
                }
            });
        }

        // 动画切换按钮
        const toggleAnimationBtn = document.getElementById('toggle-animation');
        if (toggleAnimationBtn) {
            toggleAnimationBtn.addEventListener('click', () => {
                if (this.visualizer && this.visualizer.effectsEngine) {
                    const isAnimating = this.visualizer.effectsEngine.toggleAnimation();
                    toggleAnimationBtn.textContent = isAnimating ? '暂停动画' : '恢复动画';
                }
            });
        }

        // 信息面板关闭按钮
        const closeInfoBtn = document.getElementById('close-info');
        if (closeInfoBtn) {
            closeInfoBtn.addEventListener('click', () => {
                if (this.visualizer) {
                    this.visualizer.hideInfoPanel();
                }
            });
        }

        // 搜索结果关闭按钮
        const closeSearchBtn = document.getElementById('close-search');
        if (closeSearchBtn) {
            closeSearchBtn.addEventListener('click', () => {
                this.hideSearchResults();
            });
        }
    }

    /**
     * 加载系统统计信息
     */
    async loadStats() {
        try {
            const response = await fetch('/api/stats');
            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }
            
            this.stats = await response.json();
            this.updateStatsDisplay();
            
            console.log('统计信息加载完成:', this.stats);
        } catch (error) {
            console.error('加载统计信息失败:', error);
            // 显示默认值
            this.updateStatsDisplay({
                totalTranscripts: 0,
                geminiEmbeddings: 0,
                openaiEmbeddings: 0
            });
        }
    }

    /**
     * 更新统计显示
     */
    updateStatsDisplay(stats = this.stats) {
        if (!stats) return;

        const totalCountEl = document.getElementById('total-count');
        const embeddingCountEl = document.getElementById('embedding-count');
        const coverageEl = document.getElementById('coverage');

        if (totalCountEl) {
            totalCountEl.textContent = stats.totalTranscripts.toLocaleString();
        }

        if (embeddingCountEl) {
            const totalEmbeddings = (stats.geminiEmbeddings || 0) + (stats.openaiEmbeddings || 0);
            embeddingCountEl.textContent = totalEmbeddings.toLocaleString();
        }

        if (coverageEl) {
            const coverage = stats.totalTranscripts > 0 
                ? ((stats.geminiEmbeddings || 0) / stats.totalTranscripts * 100)
                : 0;
            coverageEl.textContent = coverage.toFixed(1) + '%';
        }
    }

    /**
     * 加载默认数据
     */
    async loadDefaultData() {
        const defaultProvider = 'gemini';
        const defaultLimit = 100;
        
        this.showLoading('正在加载嵌入数据...');
        
        try {
            await this.visualizer.loadEmbeddings(defaultProvider, defaultLimit);
            
            // 更新 UI 状态
            const providerSelect = document.getElementById('provider');
            if (providerSelect) {
                providerSelect.value = defaultProvider;
            }
            
        } catch (error) {
            console.error('加载默认数据失败:', error);
            this.showError('加载数据失败: ' + error.message);
        }
    }

    /**
     * 切换提供商
     */
    async changeProvider(provider) {
        if (this.isLoading) return;
        
        this.showLoading(`正在加载 ${provider} 数据...`);
        
        try {
            await this.visualizer.loadEmbeddings(provider, 100);
            this.hideLoading();
        } catch (error) {
            console.error('切换提供商失败:', error);
            this.showError('切换提供商失败: ' + error.message);
        }
    }

    /**
     * 切换聚类方法
     */
    async changeClusterMethod(method) {
        if (this.isLoading || !this.visualizer || this.visualizer.embeddings.length === 0) {
            return;
        }
        
        this.showLoading(`正在应用 ${method.toUpperCase()} 降维...`);
        
        try {
            await this.visualizer.visualizeEmbeddings(method);
            this.hideLoading();
        } catch (error) {
            console.error('切换聚类方法失败:', error);
            this.showError('切换聚类方法失败: ' + error.message);
        }
    }

    /**
     * 执行搜索 - 增强版本
     */
    async performSearch(query) {
        // 输入验证
        const trimmedQuery = query.trim();
        if (!trimmedQuery) {
            this.showTemporaryMessage('请输入搜索关键词', 2000);
            return;
        }

        if (trimmedQuery.length < 2) {
            this.showTemporaryMessage('搜索关键词至少需要2个字符', 2000);
            return;
        }

        if (trimmedQuery.length > 200) {
            this.showTemporaryMessage('搜索关键词不能超过200个字符', 2000);
            return;
        }

        if (this.isLoading || !this.visualizer) {
            if (this.isLoading) {
                this.showTemporaryMessage('正在处理中，请稍等...', 2000);
            } else {
                this.showTemporaryMessage('可视化系统未准备好，请稍等...', 2000);
            }
            return;
        }

        this.showLoading(`正在搜索 "${trimmedQuery}"...`);

        try {
            const results = await this.visualizer.searchSimilar(trimmedQuery);
            this.displaySearchResults(results, trimmedQuery);
            this.hideLoading();
        } catch (error) {
            console.error('搜索失败:', error);
            this.showError('搜索失败: ' + error.message);
        }
    }

    /**
     * 显示搜索结果
     */
    displaySearchResults(results, query) {
        const searchResults = document.getElementById('search-results');
        const searchList = document.getElementById('search-list');
        
        if (!searchResults || !searchList) return;

        // 清空之前的结果
        searchList.innerHTML = '';

        if (results.length === 0) {
            searchList.innerHTML = `
                <div class="search-item">
                    <div class="search-item-text">没有找到与 "${query}" 相关的结果</div>
                </div>
            `;
        } else {
            results.forEach(result => {
                const item = document.createElement('div');
                item.className = 'search-item';
                item.innerHTML = `
                    <div class="search-item-id">ID: ${result.id}</div>
                    <div class="search-item-user">用户: ${result.user || '未知'}</div>
                    <div class="search-item-text">${result.textPreview || result.text || '无预览'}</div>
                `;
                
                // 点击跳转到对应粒子
                item.addEventListener('click', () => {
                    this.focusOnResult(result);
                    this.hideSearchResults();
                });
                
                searchList.appendChild(item);
            });
        }

        // 显示搜索结果面板
        searchResults.classList.remove('hidden');
    }

    /**
     * 聚焦到搜索结果
     */
    focusOnResult(result) {
        if (!this.visualizer) return;

        const particle = this.visualizer.particles.find(p => 
            p.userData.embeddingData && p.userData.embeddingData.id === result.id
        );

        if (particle) {
            this.visualizer.selectParticle(particle);
            
            // 移动相机到粒子位置
            const targetPosition = particle.position.clone();
            targetPosition.add(new THREE.Vector3(10, 10, 10));
            
            this.animateCameraTo(targetPosition, particle.position);
        }
    }

    /**
     * 动画移动相机
     */
    animateCameraTo(position, lookAt) {
        if (!this.visualizer) return;

        const camera = this.visualizer.camera;
        const controls = this.visualizer.controls;
        
        const startPosition = camera.position.clone();
        const startLookAt = controls.target.clone();
        
        let progress = 0;
        const duration = 2000; // 2秒
        const startTime = Date.now();
        
        const animate = () => {
            const elapsed = Date.now() - startTime;
            progress = Math.min(elapsed / duration, 1);
            
            // 使用缓动函数
            const easeProgress = this.easeInOutCubic(progress);
            
            // 插值相机位置
            camera.position.lerpVectors(startPosition, position, easeProgress);
            
            // 插值视角目标
            controls.target.lerpVectors(startLookAt, lookAt, easeProgress);
            
            if (progress < 1) {
                requestAnimationFrame(animate);
            } else {
                controls.update();
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
     * 隐藏搜索结果
     */
    hideSearchResults() {
        const searchResults = document.getElementById('search-results');
        if (searchResults) {
            searchResults.classList.add('hidden');
        }
    }

    /**
     * 显示加载状态
     */
    showLoading(message = '加载中...') {
        this.isLoading = true;
        
        const loading = document.getElementById('loading');
        const loadingText = document.getElementById('loading-text');
        
        if (loading) {
            loading.style.display = 'flex';
        }
        
        if (loadingText) {
            loadingText.textContent = message;
        }
    }

    /**
     * 隐藏加载状态
     */
    hideLoading() {
        this.isLoading = false;
        
        const loading = document.getElementById('loading');
        if (loading) {
            loading.style.display = 'none';
        }
    }

    /**
     * 显示错误信息
     */
    showError(message) {
        this.hideLoading();
        
        // 简单的错误提示，可以后续替换为更好的 UI
        alert('错误: ' + message);
        
        console.error('应用错误:', message);
    }

    /**
     * 显示临时消息
     */
    showTemporaryMessage(message, duration = 3000) {
        // 创建临时消息元素
        const messageDiv = document.createElement('div');
        messageDiv.textContent = message;
        messageDiv.style.cssText = `
            position: fixed;
            top: 20px;
            right: 20px;
            background: rgba(255, 217, 61, 0.9);
            color: #000;
            padding: 12px 20px;
            border-radius: 8px;
            z-index: 10000;
            font-size: 14px;
            box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
            transition: opacity 0.3s ease;
        `;
        
        document.body.appendChild(messageDiv);
        
        // 自动移除
        setTimeout(() => {
            messageDiv.style.opacity = '0';
            setTimeout(() => {
                if (messageDiv.parentNode) {
                    messageDiv.parentNode.removeChild(messageDiv);
                }
            }, 300);
        }, duration);
    }
    
    /**
     * 设置敏感度控制
     */
    setupSensitivityControls() {
        // 设备指示器和手动切换
        this.deviceIndicator = document.getElementById('input-device-indicator');
        this.deviceOverrideSelect = document.getElementById('device-override');
        this.deviceOverride = 'auto'; // 'auto', 'mouse', 'trackpad'
        
        // 鼠标敏感度控制
        const mouseSensitivitySlider = document.getElementById('mouse-sensitivity');
        const mouseSensitivityValue = document.getElementById('mouse-sensitivity-value');
        const mouseControls = mouseSensitivitySlider ? mouseSensitivitySlider.closest('.control-group') : null;
        
        // 触控板敏感度控制
        const trackpadSensitivitySlider = document.getElementById('trackpad-sensitivity');
        const trackpadSensitivityValue = document.getElementById('trackpad-sensitivity-value');
        const trackpadControls = trackpadSensitivitySlider ? trackpadSensitivitySlider.closest('.control-group') : null;
        
        // 存储控制元素引用
        this.sensitivityControls = {
            mouse: { slider: mouseSensitivitySlider, value: mouseSensitivityValue, group: mouseControls },
            trackpad: { slider: trackpadSensitivitySlider, value: trackpadSensitivityValue, group: trackpadControls }
        };
        
        // 设置鼠标敏感度控制
        if (mouseSensitivitySlider && mouseSensitivityValue) {
            mouseSensitivitySlider.addEventListener('input', (e) => {
                const sensitivity = parseFloat(e.target.value);
                mouseSensitivityValue.textContent = sensitivity.toFixed(1);
                if (this.visualizer) {
                    this.visualizer.setSensitivityMultiplier('mouse', sensitivity);
                }
            });
        }
        
        // 设置触控板敏感度控制
        if (trackpadSensitivitySlider && trackpadSensitivityValue) {
            trackpadSensitivitySlider.addEventListener('input', (e) => {
                const sensitivity = parseFloat(e.target.value);
                trackpadSensitivityValue.textContent = sensitivity.toFixed(1);
                if (this.visualizer) {
                    this.visualizer.setSensitivityMultiplier('trackpad', sensitivity);
                }
            });
        }
        
        // 手动设备切换
        if (this.deviceOverrideSelect) {
            this.deviceOverrideSelect.addEventListener('change', (e) => {
                this.deviceOverride = e.target.value;
                if (this.deviceOverride !== 'auto' && this.visualizer) {
                    this.visualizer.setDeviceType(this.deviceOverride);
                }
                this.updateDeviceIndicator();
            });
        }
        
        // 定期更新设备检测状态
        this.deviceDetectionInterval = setInterval(() => {
            this.updateDeviceIndicator();
        }, 1000);
    }
    
    /**
     * 更新设备指示器
     */
    updateDeviceIndicator() {
        if (!this.visualizer || !this.deviceIndicator) return;
        
        let currentDevice, confidence;
        
        // 检查是否手动覆盖
        if (this.deviceOverride !== 'auto') {
            currentDevice = this.deviceOverride;
            confidence = 1.0;
        } else {
            currentDevice = this.visualizer.inputDevice;
            confidence = this.visualizer.trackpadConfidence;
            
            // 如果长时间未检测成功，显示提示
            if (currentDevice === 'unknown' && this.detectionStartTime && 
                Date.now() - this.detectionStartTime > 5000) {
                this.deviceIndicator.textContent = '检测失败 - 请手动选择';
                this.deviceIndicator.className = 'device-indicator detecting';
                return;
            } else if (currentDevice === 'unknown' && !this.detectionStartTime) {
                this.detectionStartTime = Date.now();
            } else if (currentDevice !== 'unknown') {
                this.detectionStartTime = null;
            }
        }
        
        // 更新指示器文本和样式
        this.deviceIndicator.className = 'device-indicator';
        
        if (currentDevice === 'unknown' || (this.deviceOverride === 'auto' && confidence < 0.3)) {
            this.deviceIndicator.textContent = '检测中...';
            this.deviceIndicator.classList.add('detecting');
        } else if (currentDevice === 'mouse') {
            this.deviceIndicator.textContent = this.deviceOverride !== 'auto' ? '鼠标 (手动)' : '鼠标';
            this.deviceIndicator.classList.add('mouse');
        } else if (currentDevice === 'trackpad') {
            this.deviceIndicator.textContent = this.deviceOverride !== 'auto' ? '触控板 (手动)' : '触控板';
            this.deviceIndicator.classList.add('trackpad');
        }
        
        // 高亮当前活跃的敏感度控制
        Object.keys(this.sensitivityControls).forEach(deviceType => {
            const controls = this.sensitivityControls[deviceType];
            if (controls.group) {
                if (deviceType === currentDevice) {
                    controls.group.classList.add('active');
                } else {
                    controls.group.classList.remove('active');
                }
            }
        });
    }

    dispose() {
        // 清理设备检测定时器
        if (this.deviceDetectionInterval) {
            clearInterval(this.deviceDetectionInterval);
            this.deviceDetectionInterval = null;
        }
        
        if (this.visualizer) {
            this.visualizer.dispose();
            this.visualizer = null;
        }
    }
}

// 等待 DOM 加载完成后初始化应用
document.addEventListener('DOMContentLoaded', () => {
    // 检查浏览器兼容性
    if (typeof THREE === 'undefined') {
        alert('浏览器不支持 Three.js，请使用现代浏览器访问');
        return;
    }

    // 初始化应用
    try {
        window.app = new EmbeddingApp();
        console.log('🚀 应用启动成功');
    } catch (error) {
        console.error('应用启动失败:', error);
        alert('应用启动失败: ' + error.message);
    }
});

// 在页面卸载时清理资源
window.addEventListener('beforeunload', () => {
    if (window.app) {
        window.app.dispose();
    }
});

// 导出到全局范围
window.EmbeddingApp = EmbeddingApp;