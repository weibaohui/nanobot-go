# 016-Provider对话模型表格化-设计

## 变更记录表

| 版本 | 日期 | 变更内容 | 作者 |
|------|------|----------|------|
| v1.0 | 2026-03-14 | 初始设计文档 | AI |

---

## 1. 设计概述

将 Provider 编辑弹窗中的对话模型配置从 List 组件改为 Table 组件，与嵌入模型配置保持一致的交互体验。

## 2. 详细设计

### 2.1 状态管理

新增状态：
```typescript
const [defaultModel, setDefaultModel] = useState(''); // 默认模型ID
const [modelInput, setModelInput] = useState({ id: '', name: '', maxTokens: 8192 }); // 添加表单
```

### 2.2 核心方法

新增方法：
```typescript
// 设置默认模型
const handleSetDefaultModel = (modelId: string) => {
  setDefaultModel(modelId);
  message.success('已设为默认模型');
};

// 添加模型（含重复校验）
const handleAddModel = () => {
  if (!modelInput.id || !modelInput.name) {
    message.warning('请输入模型 ID 和名称');
    return;
  }
  if (models.some(m => m.id === modelInput.id)) {
    message.warning('模型 ID 已存在');
    return;
  }
  setModels([...models, { ...modelInput }]);
  setModelInput({ id: '', name: '', maxTokens: 8192 });
};

// 删除模型（如果是默认模型则清除）
const handleRemoveModel = (index: number) => {
  const model = models[index];
  if (model.id === defaultModel) {
    setDefaultModel('');
  }
  setModels(models.filter((_, i) => i !== index));
};
```

### 2.3 表格列定义

```typescript
const modelColumns = [
  { title: '模型ID', dataIndex: 'id', ellipsis: true },
  { title: '名称', dataIndex: 'name' },
  { title: '最大Token', dataIndex: 'max_tokens', width: 120 },
  {
    title: '默认',
    width: 80,
    render: (_: any, record: ModelInfo) =>
      record.id === defaultModel ? <Tag color="gold">默认</Tag> : null,
  },
  {
    title: '操作',
    width: 180,
    render: (_: any, record: ModelInfo, index: number) => (
      <Space size="small">
        {record.id !== defaultModel && (
          <Button
            type="text"
            size="small"
            icon={<StarOutlined />}
            onClick={() => handleSetDefaultModel(record.id)}
          >
            设为默认
          </Button>
        )}
        <Button
          type="text"
          size="small"
          danger
          onClick={() => handleRemoveModel(index)}
        >
          删除
        </Button>
      </Space>
    ),
  },
];
```

### 2.4 UI 布局

```
支持的模型:

+-----------------------------------------------+
| 模型ID    | 名称    | 最大Token | 默认 | 操作 |
+-----------------------------------------------+
| claude-3  | Claude3 | 200000    | 默认 | 删除 |
| gpt-4     | GPT-4   | 128000    | -    | 设为默认 删除 |
+-----------------------------------------------+

添加新模型:
+-----------------------------------------------+
| [模型ID      ] [模型名称      ] [最大Token ] [+] |
+-----------------------------------------------+
```

## 3. 数据处理

### 3.1 打开弹窗时加载
```typescript
const openModal = (provider?: LLMProvider) => {
  if (provider) {
    const supportedModels = provider.supported_models ? JSON.parse(provider.supported_models) : [];
    setModels(supportedModels);
    setDefaultModel(provider.default_model || ''); // 加载默认模型
  }
};
```

### 3.2 保存时处理
```typescript
const handleUpdate = async (values: CreateProviderRequest) => {
  await providersApi.update(editingProvider.id, {
    ...values,
    supported_models: models,
    default_model: defaultModel, // 保存默认模型
  });
};
```

## 4. 样式规范

- 表格使用 `size="small"`
- 添加区域使用灰色背景 `background: '#f5f5f5'`
- 圆角 `borderRadius: 8`
- 内边距 `padding: 16`

## 5. 错误处理

- 模型ID重复提示警告
- 必填字段校验
- 删除默认模型时自动清除默认设置
