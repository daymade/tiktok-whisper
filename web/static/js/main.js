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

        // 搜索功能
        const searchInput = document.getElementById('search-input');
        const searchBtn = document.getElementById('search-btn');
        
        if (searchInput && searchBtn) {
            searchBtn.addEventListener('click', () => {
                this.performSearch(searchInput.value);
            });
            
            searchInput.addEventListener('keypress', (e) => {
                if (e.key === 'Enter') {
                    this.performSearch(searchInput.value);
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
     * 执行搜索
     */
    async performSearch(query) {
        if (!query.trim()) {
            this.showError('请输入搜索关键词');
            return;
        }

        if (this.isLoading || !this.visualizer) {
            return;
        }

        this.showLoading(`正在搜索 "${query}"...`);

        try {
            const results = await this.visualizer.searchSimilar(query);
            this.displaySearchResults(results, query);
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
     * 销毁应用
     */
    dispose() {
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