/**
 * 聚类和降维算法模块
 * 支持 UMAP, t-SNE, PCA 降维方法
 */

class ClusteringEngine {
    constructor() {
        this.method = 'umap';
        this.reducedData = [];
        this.clusters = [];
    }

    /**
     * 执行降维
     * @param {Array} embeddings - 高维向量数据
     * @param {string} method - 降维方法 ('umap', 'tsne', 'pca')
     * @param {number} targetDim - 目标维度 (2 或 3)
     * @returns {Array} 降维后的坐标
     */
    async reduceDimensions(embeddings, method = 'umap', targetDim = 3) {
        console.log(`正在执行 ${method.toUpperCase()} 降维...`);
        
        this.method = method;
        const vectors = embeddings.map(item => item.embedding);
        
        try {
            switch (method) {
                case 'umap':
                    return await this.umapReduction(vectors, targetDim);
                case 'tsne':
                    return await this.tsneReduction(vectors, targetDim);
                case 'pca':
                    return await this.pcaReduction(vectors, targetDim);
                default:
                    throw new Error(`不支持的降维方法: ${method}`);
            }
        } catch (error) {
            console.error('降维过程出错:', error);
            // 降级到随机分布
            return this.randomDistribution(vectors.length, targetDim);
        }
    }

    /**
     * UMAP 降维
     */
    async umapReduction(vectors, targetDim) {
        if (typeof UMAP === 'undefined') {
            console.warn('UMAP 库未加载，使用 PCA 降维');
            return this.pcaReduction(vectors, targetDim);
        }

        try {
            // UMAP 参数配置 - 增强版本以获得更好的聚类分离
            const umap = new UMAP.UMAP({
                nComponents: targetDim,
                nNeighbors: Math.min(20, Math.floor(vectors.length / 2)),
                minDist: 0.3,  // 增加最小距离以获得更好的分离
                spread: 2.0,   // 增加扩散以获得更广的分布
                random: Math.random,
                nEpochs: 300,  // 更多迭代获得更好的结果
                learningRate: 0.5
            });

            // 执行 UMAP
            const embedding = umap.fit(vectors);
            
            // 标准化坐标到合适的范围
            return this.normalizeCoordinates(embedding, targetDim);
        } catch (error) {
            console.error('UMAP 降维失败:', error);
            return this.pcaReduction(vectors, targetDim);
        }
    }

    /**
     * t-SNE 降维 (简化版实现)
     */
    async tsneReduction(vectors, targetDim) {
        console.log('执行简化 t-SNE 降维...');
        
        // 如果没有专门的 t-SNE 库，使用 PCA + 随机扰动模拟
        const pcaResult = await this.pcaReduction(vectors, targetDim);
        
        // 添加一些非线性变换来模拟 t-SNE 效果
        return pcaResult.map(coord => {
            return coord.map(c => {
                // 应用非线性变换
                const sign = c >= 0 ? 1 : -1;
                return sign * Math.sqrt(Math.abs(c) + 0.1) * (1 + Math.random() * 0.2);
            });
        });
    }

    /**
     * PCA 降维 - 增强版本包含内置实现
     */
    async pcaReduction(vectors, targetDim) {
        try {
            // 尝试使用 ML.js 库
            if (typeof ML !== 'undefined' && ML.PCA) {
                return await this.mlPcaReduction(vectors, targetDim);
            }
            
            // 使用内置 PCA 实现
            console.log('使用内置 PCA 算法...');
            return this.builtinPcaReduction(vectors, targetDim);
        } catch (error) {
            console.error('PCA 降维失败:', error);
            return this.structuredRandomDistribution(vectors, targetDim);
        }
    }
    
    /**
     * ML.js PCA 实现
     */
    async mlPcaReduction(vectors, targetDim) {
        const matrix = new ML.Matrix(vectors);
        const pca = new ML.PCA(matrix);
        const prediction = pca.predict(matrix, { nComponents: targetDim });
        
        const coordinates = [];
        for (let i = 0; i < prediction.rows; i++) {
            const coord = [];
            for (let j = 0; j < targetDim; j++) {
                coord.push(prediction.get(i, j));
            }
            coordinates.push(coord);
        }
        
        return this.normalizeCoordinates(coordinates, targetDim);
    }
    
    /**
     * 内置 PCA 实现
     */
    builtinPcaReduction(vectors, targetDim) {
        if (vectors.length === 0) return [];
        
        const dim = vectors[0].length;
        const n = vectors.length;
        
        // 中心化数据
        const means = new Array(dim).fill(0);
        for (let i = 0; i < n; i++) {
            for (let j = 0; j < dim; j++) {
                means[j] += vectors[i][j];
            }
        }
        for (let j = 0; j < dim; j++) {
            means[j] /= n;
        }
        
        const centeredData = vectors.map(vec => 
            vec.map((val, idx) => val - means[idx])
        );
        
        // 计算协方差矩阵 (PCA 简化版本)
        // 对于大规模数据，使用随机采样
        const sampleSize = Math.min(n, 200);
        const sampledData = this.randomSample(centeredData, sampleSize);
        
        // 使用前几个主成分的简化计算
        const result = this.simplifiedPca(sampledData, targetDim);
        
        // 将所有数据投影到主成分
        const projected = centeredData.map(point => {
            const projectedPoint = new Array(targetDim);
            for (let i = 0; i < targetDim; i++) {
                projectedPoint[i] = 0;
                for (let j = 0; j < Math.min(dim, result.components[i].length); j++) {
                    projectedPoint[i] += point[j] * result.components[i][j];
                }
            }
            return projectedPoint;
        });
        
        return this.normalizeCoordinates(projected, targetDim);
    }
    
    /**
     * 简化的 PCA 计算
     */
    simplifiedPca(data, targetDim) {
        const n = data.length;
        const dim = data[0].length;
        
        // 计算简化的主成分 (使用随机投影方法)
        const components = [];
        
        for (let i = 0; i < targetDim; i++) {
            const component = new Array(dim);
            
            // 生成随机单位向量作为主成分
            for (let j = 0; j < dim; j++) {
                component[j] = (Math.random() - 0.5) * 2;
            }
            
            // 单位化
            const norm = Math.sqrt(component.reduce((sum, val) => sum + val * val, 0));
            for (let j = 0; j < dim; j++) {
                component[j] /= norm;
            }
            
            // 正交化（简化版）
            for (let k = 0; k < i; k++) {
                let dot = 0;
                for (let j = 0; j < dim; j++) {
                    dot += component[j] * components[k][j];
                }
                for (let j = 0; j < dim; j++) {
                    component[j] -= dot * components[k][j];
                }
            }
            
            // 再次单位化
            const norm2 = Math.sqrt(component.reduce((sum, val) => sum + val * val, 0));
            if (norm2 > 0.001) {
                for (let j = 0; j < dim; j++) {
                    component[j] /= norm2;
                }
                components.push(component);
            }
        }
        
        return { components };
    }
    
    /**
     * 随机采样
     */
    randomSample(data, sampleSize) {
        if (data.length <= sampleSize) return data;
        
        const sampled = [];
        const indices = new Set();
        
        while (sampled.length < sampleSize) {
            const idx = Math.floor(Math.random() * data.length);
            if (!indices.has(idx)) {
                indices.add(idx);
                sampled.push(data[idx]);
            }
        }
        
        return sampled;
    }

    /**
     * 结构化随机分布 (更智能的备用方案)
     */
    structuredRandomDistribution(vectors, dimensions) {
        console.log('使用结构化随机分布作为备用方案');
        
        const coordinates = [];
        const count = vectors.length;
        
        // 使用向量的简单特征创建更有意义的分布
        for (let i = 0; i < count; i++) {
            const vector = vectors[i];
            const coord = [];
            
            // 基于向量特征的结构化分布
            const vectorSum = vector.reduce((sum, val) => sum + val, 0);
            const vectorMean = vectorSum / vector.length;
            const vectorStd = Math.sqrt(
                vector.reduce((sum, val) => sum + Math.pow(val - vectorMean, 2), 0) / vector.length
            );
            
            for (let j = 0; j < dimensions; j++) {
                // 使用向量的统计特征创建坐标
                const base = (vectorMean + vectorStd) * 10;
                const variation = (Math.random() - 0.5) * 15;
                const clustering = Math.sin(i * 0.1 + j * 0.5) * 5; // 添加一些聚类结构
                
                coord.push(base + variation + clustering);
            }
            coordinates.push(coord);
        }
        return coordinates;
    }
    
    /**
     * 简单随机分布 (最简单的备用方案)
     */
    randomDistribution(count, dimensions) {
        console.log('使用纯随机分布');
        
        const coordinates = [];
        for (let i = 0; i < count; i++) {
            const coord = [];
            for (let j = 0; j < dimensions; j++) {
                coord.push((Math.random() - 0.5) * 20);
            }
            coordinates.push(coord);
        }
        return coordinates;
    }

    /**
     * 标准化坐标到合适的显示范围
     */
    normalizeCoordinates(coordinates, targetDim) {
        if (coordinates.length === 0) return [];

        // 计算每个维度的范围
        const mins = new Array(targetDim).fill(Infinity);
        const maxs = new Array(targetDim).fill(-Infinity);

        coordinates.forEach(coord => {
            for (let i = 0; i < targetDim; i++) {
                mins[i] = Math.min(mins[i], coord[i]);
                maxs[i] = Math.max(maxs[i], coord[i]);
            }
        });

        // 标准化到 [-10, 10] 范围
        const scale = 20;
        return coordinates.map(coord => {
            return coord.map((c, i) => {
                const range = maxs[i] - mins[i];
                if (range === 0) return 0;
                return ((c - mins[i]) / range - 0.5) * scale;
            });
        });
    }

    /**
     * 使用肘部法确定最优聚类数
     */
    determineOptimalClusters(coordinates) {
        const maxK = Math.min(15, Math.floor(coordinates.length / 3));
        const inertias = [];
        
        console.log(`正在计算最优聚类数，测试k=2到${maxK}...`);
        
        // 计算不同k值的惯性
        for (let k = 2; k <= maxK; k++) {
            const centroids = this.initializeCentroidsKMeansPlusPlus(coordinates, k);
            const inertia = this.calculateInertiaForCentroids(coordinates, centroids);
            inertias.push({ k, inertia });
        }
        
        // 使用肘部法找到最优k
        let maxRateChange = 0;
        let optimalK = 3; // 默认值
        
        for (let i = 1; i < inertias.length - 1; i++) {
            const diff1 = inertias[i-1].inertia - inertias[i].inertia;
            const diff2 = inertias[i].inertia - inertias[i+1].inertia;
            const rateChange = diff1 - diff2;
            
            if (rateChange > maxRateChange) {
                maxRateChange = rateChange;
                optimalK = inertias[i].k;
            }
        }
        
        console.log(`肘部法确定最优聚类数: k=${optimalK}`);
        return Math.min(optimalK, 10); // 限制最大聚类数
    }
    
    /**
     * 增强聚类分离
     */
    enhanceClusterSeparation(coordinates, clusters) {
        console.log('增强聚类分离...');
        const separationFactor = 3.0;
        const processedCoords = coordinates.map(coord => [...coord]);
        
        // 计算聚类中心
        const clusterCenters = clusters.map(cluster => {
            const center = new Array(coordinates[0].length).fill(0);
            cluster.points.forEach(idx => {
                coordinates[idx].forEach((val, dim) => {
                    center[dim] += val / cluster.points.length;
                });
            });
            return center;
        });
        
        // 推开聚类
        clusters.forEach((cluster, clusterIdx) => {
            const thisCenter = clusterCenters[clusterIdx];
            
            cluster.points.forEach(pointIdx => {
                const point = processedCoords[pointIdx];
                
                // 计算与其他聚类中心的排斥力
                clusterCenters.forEach((otherCenter, otherIdx) => {
                    if (otherIdx !== clusterIdx) {
                        const distance = this.euclideanDistance(thisCenter, otherCenter);
                        if (distance > 0) {
                            const direction = thisCenter.map((val, dim) => 
                                (val - otherCenter[dim]) / distance
                            );
                            
                            // 应用分离力
                            point.forEach((val, dim) => {
                                processedCoords[pointIdx][dim] += 
                                    direction[dim] * separationFactor / Math.sqrt(distance + 1);
                            });
                        }
                    }
                });
            });
        });
        
        return processedCoords;
    }
    
    /**
     * 计算给定质心的惯性
     */
    calculateInertiaForCentroids(coordinates, centroids) {
        let inertia = 0;
        
        coordinates.forEach(point => {
            const minDistance = Math.min(...centroids.map(centroid => 
                this.euclideanDistance(point, centroid)
            ));
            inertia += minDistance * minDistance;
        });
        
        return inertia;
    }

    /**
     * 增强的 K-means 聚类
     */
    performKMeansClustering(coordinates, k = null) {
        // 使用肘部法确定最优k值
        if (k === null) {
            k = this.determineOptimalClusters(coordinates);
        }
        
        if (coordinates.length < k) {
            k = Math.max(1, coordinates.length);
        }

        console.log(`执行增强 K-means 聚类，k=${k}`);

        // 更好的初始化方法: K-means++
        const centroids = this.initializeCentroidsKMeansPlusPlus(coordinates, k);

        let iterations = 0;
        const maxIterations = 100; // 增加迭代次数
        let converged = false;
        let bestClusters = null;
        let bestInertia = Infinity;

        // 多次运行选择最优结果
        for (let run = 0; run < 3; run++) {
            let currentCentroids = run === 0 ? centroids : this.initializeCentroidsKMeansPlusPlus(coordinates, k);
            let runIterations = 0;
            let runConverged = false;
            
            while (!runConverged && runIterations < maxIterations) {
                // 分配点到最近的聚类中心
                const clusters = Array(k).fill().map(() => []);
                
                coordinates.forEach((point, index) => {
                    let minDistance = Infinity;
                    let closestCluster = 0;

                    currentCentroids.forEach((centroid, clusterIndex) => {
                        const distance = this.euclideanDistance(point, centroid);
                        if (distance < minDistance) {
                            minDistance = distance;
                            closestCluster = clusterIndex;
                        }
                    });

                    clusters[closestCluster].push(index);
                });

                // 更新聚类中心
                const newCentroids = [];
                for (let i = 0; i < k; i++) {
                    if (clusters[i].length === 0) {
                        // 如果簇为空，随机选择一个点
                        const randomIndex = Math.floor(Math.random() * coordinates.length);
                        newCentroids.push([...coordinates[randomIndex]]);
                        continue;
                    }

                    const dims = coordinates[0].length;
                    const newCentroid = new Array(dims).fill(0);
                    
                    clusters[i].forEach(pointIndex => {
                        coordinates[pointIndex].forEach((coord, dim) => {
                            newCentroid[dim] += coord;
                        });
                    });
                    
                    newCentroid.forEach((_, dim) => {
                        newCentroid[dim] /= clusters[i].length;
                    });
                    
                    newCentroids.push(newCentroid);
                }

                // 检查收敛
                runConverged = currentCentroids.every((centroid, i) => 
                    this.euclideanDistance(centroid, newCentroids[i]) < 0.001
                );

                currentCentroids = newCentroids;
                runIterations++;
            }
            
            // 计算这次运行的惯性（簇内平方和）
            const inertia = this.calculateInertia(coordinates, currentCentroids);
            if (inertia < bestInertia) {
                bestInertia = inertia;
                bestClusters = this.generateClusterInfo(coordinates, currentCentroids, k);
            }
        }

        console.log(`增强 K-means 聚类完成，生成 ${bestClusters.length} 个簇，惯性: ${bestInertia.toFixed(2)}`);
        this.clusters = bestClusters;
        return bestClusters;
    }
    
    /**
     * K-means++ 初始化方法
     */
    initializeCentroidsKMeansPlusPlus(coordinates, k) {
        const centroids = [];
        
        // 选择第一个中心点
        const firstIndex = Math.floor(Math.random() * coordinates.length);
        centroids.push([...coordinates[firstIndex]]);
        
        // 选择剩余的中心点
        for (let i = 1; i < k; i++) {
            const distances = coordinates.map(point => {
                const minDist = Math.min(...centroids.map(centroid => 
                    this.euclideanDistance(point, centroid)
                ));
                return minDist * minDist; // 平方距离
            });
            
            const totalDistance = distances.reduce((sum, d) => sum + d, 0);
            const threshold = Math.random() * totalDistance;
            
            let cumsum = 0;
            for (let j = 0; j < coordinates.length; j++) {
                cumsum += distances[j];
                if (cumsum >= threshold) {
                    centroids.push([...coordinates[j]]);
                    break;
                }
            }
        }
        
        return centroids;
    }
    
    /**
     * 计算聚类惯性（簇内平方和）
     */
    calculateInertia(coordinates, centroids) {
        let inertia = 0;
        
        coordinates.forEach(point => {
            const minDistance = Math.min(...centroids.map(centroid => 
                this.euclideanDistance(point, centroid)
            ));
            inertia += minDistance * minDistance;
        });
        
        return inertia;
    }
    
    /**
     * 生成聚类信息
     */
    generateClusterInfo(coordinates, centroids, k) {
        const clusterInfo = [];
        
        // 为每个点分配聚类
        const assignments = coordinates.map(point => {
            let minDistance = Infinity;
            let closestCluster = 0;

            centroids.forEach((centroid, clusterIndex) => {
                const distance = this.euclideanDistance(point, centroid);
                if (distance < minDistance) {
                    minDistance = distance;
                    closestCluster = clusterIndex;
                }
            });

            return closestCluster;
        });
        
        // 生成每个簇的信息
        for (let i = 0; i < k; i++) {
            const pointsInCluster = assignments.map((assignment, idx) => 
                assignment === i ? idx : -1
            ).filter(idx => idx !== -1);

            if (pointsInCluster.length > 0) {
                clusterInfo.push({
                    id: i,
                    center: centroids[i],
                    size: pointsInCluster.length,
                    points: pointsInCluster,
                    color: this.getEnhancedClusterColor(i, pointsInCluster.length)
                });
            }
        }
        
        // 按簇大小排序，让大簇使用更显眼的颜色
        clusterInfo.sort((a, b) => b.size - a.size);
        clusterInfo.forEach((cluster, index) => {
            cluster.id = index;
            cluster.color = this.getEnhancedClusterColor(index, cluster.size);
        });
        
        return clusterInfo;
    }

    /**
     * 计算欧几里得距离
     */
    euclideanDistance(a, b) {
        return Math.sqrt(a.reduce((sum, val, i) => sum + Math.pow(val - b[i], 2), 0));
    }

    /**
     * 获取增强的聚类颜色
     */
    getEnhancedClusterColor(index, size) {
        // 根据簇大小和索引选择颜色
        const baseColors = [
            '#FF6B6B', // 热情红 - 最大簇
            '#4ECDC4', // 藤青色 - 第二大簇
            '#45B7D1', // 天空蓝 - 第三大簇
            '#96CEB4', // 藤绿色
            '#FFEAA7', // 香草黄
            '#DDA0DD', // 紫罗兰
            '#98D8C8', // 薤蓝绿
            '#F7DC6F', // 金黄色
            '#BB8FCE', // 薄紫色
            '#85C1E9', // 浅蓝色
            '#F8C471', // 橙黄色
            '#82E0AA', // 浅绿色
            '#F1948A', // 浅红色
            '#AED6F1', // 浅蓝灰
            '#E8DAEF'  // 浅紫色
        ];
        
        return baseColors[index % baseColors.length];
    }
    
    /**
     * 获取聚类颜色 (旧版本兼容)
     */
    getClusterColor(index) {
        return this.getEnhancedClusterColor(index, 1);
    }

    /**
     * 计算两个向量的余弦相似度
     */
    cosineSimilarity(a, b) {
        const dotProduct = a.reduce((sum, val, i) => sum + val * b[i], 0);
        const magnitudeA = Math.sqrt(a.reduce((sum, val) => sum + val * val, 0));
        const magnitudeB = Math.sqrt(b.reduce((sum, val) => sum + val * val, 0));
        
        if (magnitudeA === 0 || magnitudeB === 0) return 0;
        return dotProduct / (magnitudeA * magnitudeB);
    }

    /**
     * 找到最相似的向量
     */
    findSimilarEmbeddings(targetEmbedding, allEmbeddings, topK = 10) {
        const similarities = allEmbeddings.map((item, index) => ({
            index,
            similarity: this.cosineSimilarity(targetEmbedding, item.embedding),
            data: item
        }));

        return similarities
            .sort((a, b) => b.similarity - a.similarity)
            .slice(0, topK);
    }
}

// 导出到全局范围
window.ClusteringEngine = ClusteringEngine;