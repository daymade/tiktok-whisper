# 🌟 抖音转录数据 Embedding 可视化系统

一个炫酷的3D交互式embedding可视化平台，专为抖音转录数据的语义分析而设计。

## ✨ 主要特性

### 🎨 炫酷的3D可视化
- **粒子系统**: 每个embedding以发光粒子形式展示
- **星空背景**: 动态星空背景营造沉浸式体验
- **光晕效果**: 粒子周围的柔和光晕和脉动动画
- **平滑动画**: 流畅的相机移动和转场效果

### 🧠 智能聚类分析
- **多种降维算法**: UMAP、t-SNE、PCA
- **自动聚类**: K-means聚类自动发现语义相似的内容
- **颜色编码**: 不同聚类用不同颜色标识
- **动态连接**: 同类内容之间的动态连接线

### 🔍 实时搜索功能
- **文本搜索**: 输入关键词实时搜索相似内容
- **涟漪效果**: 搜索结果用黄色涟漪标记
- **相似度评分**: 显示内容相似度分数
- **智能高亮**: 相关内容自动高亮显示

### 🎮 丰富的交互功能
- **鼠标控制**: 旋转、缩放、平移视角
- **点击详情**: 点击粒子查看详细信息
- **悬停预览**: 悬停显示内容预览
- **键盘快捷键**: 
  - `Space`: 暂停/恢复动画
  - `R`: 重置视角
  - `Esc`: 取消选择

## 🚀 快速开始

### 1. 启动web服务器
```bash
# 默认端口8080
./v2t web

# 指定端口
./v2t web --port :9000
```

### 2. 访问可视化界面
打开浏览器访问: `http://localhost:8080`

### 3. 操作指南
1. **选择数据源**: 在顶部选择 Gemini 或 OpenAI embeddings
2. **搜索内容**: 在搜索框输入关键词查找相似内容
3. **调整视图**: 
   - 拖拽鼠标旋转视角
   - 滚轮缩放
   - 右键拖拽平移
4. **查看详情**: 点击任意粒子查看转录详情
5. **切换算法**: 选择不同的降维算法重新布局

## 📊 数据要求

系统需要PostgreSQL数据库中包含embedding数据：

```sql
-- 需要的表结构
CREATE TABLE transcriptions (
    id SERIAL PRIMARY KEY,
    user_nickname TEXT,
    transcription TEXT,
    embedding_gemini vector(768),    -- Gemini embeddings
    embedding_openai vector(1536),  -- OpenAI embeddings
    embedding_gemini_created_at TIMESTAMP,
    embedding_openai_created_at TIMESTAMP
);
```

## 🛠 技术架构

### 后端 (Go)
- **Web框架**: 原生 `net/http`
- **数据库**: PostgreSQL + pgvector
- **API设计**: RESTful API
- **向量操作**: 高效的向量相似度计算

### 前端 (JavaScript)
- **3D引擎**: Three.js
- **降维算法**: UMAP.js, PCA
- **动画系统**: 自定义粒子效果引擎
- **UI框架**: 原生JavaScript + CSS3

## 📁 项目结构

```
web/
├── server.go              # Web服务器主文件
├── handlers/              # API处理器
│   ├── api.go            # 数据API
│   └── static.go         # 静态文件服务
└── static/               # 前端资源
    ├── index.html        # 主页面
    ├── css/style.css     # 样式文件
    └── js/               # JavaScript模块
        ├── main.js       # 应用主控制器
        ├── visualization.js # 3D可视化引擎
        ├── clustering.js # 聚类算法
        └── effects.js    # 视觉效果引擎
```

## 🎯 API端点

### 获取统计信息
```http
GET /api/stats
```

### 获取embedding数据
```http
GET /api/embeddings?provider=gemini&limit=100
```

### 搜索相似内容
```http
GET /api/embeddings/search?q=关键词&provider=gemini&limit=10
```

### 获取用户列表
```http
GET /api/users
```

## 🔧 配置选项

### 环境变量
- `OPENAI_API_KEY`: OpenAI API密钥
- `GEMINI_API_KEY`: Google Gemini API密钥

### 启动参数
- `--port`: 指定服务器端口 (默认: :8080)

## 🎨 自定义样式

系统使用CSS变量，可以轻松定制颜色主题：

```css
:root {
  --primary-color: #4ecdc4;    /* 主要颜色 */
  --secondary-color: #45b7d1;  /* 次要颜色 */
  --accent-color: #ff6b6b;     /* 强调颜色 */
  --search-color: #ffd93d;     /* 搜索高亮 */
}
```

## 🔍 故障排除

### 常见问题

1. **页面空白**
   - 检查浏览器控制台错误
   - 确认Three.js库正常加载
   - 验证WebGL支持

2. **没有数据显示**
   - 确认数据库连接正常
   - 检查embedding数据是否存在
   - 验证API端点响应

3. **性能问题**
   - 减少显示的数据点数量
   - 降低粒子效果质量
   - 关闭部分动画效果

### 浏览器兼容性
- Chrome 80+ (推荐)
- Firefox 75+
- Safari 13+
- Edge 80+

## 🚀 性能优化

- **批量渲染**: 使用实例化渲染优化大量粒子
- **LOD系统**: 距离远的粒子自动简化
- **内存管理**: 智能资源回收机制
- **渐进加载**: 大数据集分批加载

## 📈 扩展功能

未来可以添加的功能：
- 时间轴动画显示embedding演变
- 多用户协同分析
- 导出高质量可视化图像
- VR/AR支持
- 机器学习模型训练可视化

## 🤝 贡献指南

欢迎提交Issue和Pull Request来改进这个项目！

---

**享受探索你的embedding数据吧！** 🎉