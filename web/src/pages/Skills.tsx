import React, { useEffect, useState } from 'react';
import {
  Table,
  Button,
  Space,
  Tag,
  Modal,
  Card,
  Grid,
  Typography,
  Descriptions,
  Empty,
  Spin,
  message,
} from 'antd';
import { EyeOutlined } from '@ant-design/icons';
import { skillsApi } from '../api';
import type { Skill, SkillDetail } from '../types';
import type { TableColumnsType } from 'antd';

const { useBreakpoint } = Grid;
const { Title } = Typography;

const Skills: React.FC = () => {
  const screens = useBreakpoint();
  const [skills, setSkills] = useState<Skill[]>([]);
  const [loading, setLoading] = useState(false);
  const [detailVisible, setDetailVisible] = useState(false);
  const [selectedSkill, setSelectedSkill] = useState<SkillDetail | null>(null);
  const [detailLoading, setDetailLoading] = useState(false);

  // fetchSkills 获取技能列表
  // API: GET /api/v1/skills
  // 返回: { items: Skill[] }
  const fetchSkills = async () => {
    setLoading(true);
    try {
      const res = await skillsApi.list();
      setSkills((res.items || []) as Skill[]);
    } catch (error) {
      console.error('获取技能列表失败:', error);
      message.error('获取技能列表失败，请稍后重试');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchSkills();
  }, []);

  // 获取技能详情
  // API: GET /api/v1/skills/:name
  // 返回: SkillDetail (包含 bound_agents)
  const openDetail = async (skillName: string) => {
    setDetailLoading(true);
    setDetailVisible(true);
    try {
      const res = await skillsApi.get(skillName);
      setSelectedSkill(res || null);
    } catch (error) {
      console.error('获取技能详情失败:', error);
      message.error('获取技能详情失败');
      setSelectedSkill(null);
    } finally {
      setDetailLoading(false);
    }
  };

  const columns: TableColumnsType<Skill> = [
    {
      title: '名称',
      dataIndex: 'name',
      key: 'name',
      width: 200,
      render: (text: string) => (
        <Tag color="blue">{text}</Tag>
      ),
    },
    {
      title: '描述',
      dataIndex: 'description',
      key: 'description',
      ellipsis: true,
    },
    {
      title: '来源',
      dataIndex: 'source',
      key: 'source',
      width: screens.xs ? 80 : 120,
      render: (source: string) => {
        const color = source === 'workspace' ? 'green' : 'default';
        return <Tag color={color}>{source === 'workspace' ? '工作区' : '内置'}</Tag>;
      },
    },
    {
      title: '操作',
      key: 'action',
      width: screens.xs ? 80 : 100,
      render: (_: unknown, record: Skill) => (
        <Button
          type="link"
          icon={<EyeOutlined />}
          onClick={() => openDetail(record.name)}
        >
          查看详情
        </Button>
      ),
    },
  ];

  return (
    <div>
      <Card
        title={<Title level={screens.xs ? 4 : 3} style={{ margin: 0 }}>技能管理</Title>}
        styles={{ body: { padding: screens.xs ? 12 : 24 } }}
      >
        <Table
          rowKey="name"
          columns={columns}
          dataSource={skills}
          loading={loading}
          scroll={{ x: screens.xs ? 500 : undefined }}
          size={screens.xs ? 'small' : 'middle'}
          locale={{
            emptyText: '暂无技能',
          }}
        />
      </Card>

      {/* 技能详情 Modal */}
      <Modal
        title={`技能详情 - ${selectedSkill?.name || ''}`}
        open={detailVisible}
        onCancel={() => {
          setDetailVisible(false);
          setSelectedSkill(null);
        }}
        footer={null}
        width={screens.xs ? '100%' : 800}
        style={{ top: screens.xs ? 0 : 50 }}
      >
        {detailLoading ? (
          <div style={{ textAlign: 'center', padding: '40px' }}>
            <Spin />
          </div>
        ) : selectedSkill ? (
          <div>
            <Descriptions column={screens.xs ? 1 : 2} bordered>
              <Descriptions.Item label="名称">
                <Tag color="blue">{selectedSkill.name}</Tag>
              </Descriptions.Item>
              <Descriptions.Item label="来源">
                <Tag color={selectedSkill.source === 'workspace' ? 'green' : 'default'}>
                  {selectedSkill.source === 'workspace' ? '工作区' : '内置'}
                </Tag>
              </Descriptions.Item>
              <Descriptions.Item label="描述" span={screens.xs ? 1 : 2}>
                {selectedSkill.description}
              </Descriptions.Item>
            </Descriptions>

            <div style={{ marginTop: 24 }}>
              <Title level={5}>技能内容</Title>
              <div
                style={{
                  background: '#f5f5f5',
                  padding: 16,
                  borderRadius: 8,
                  fontFamily: 'monospace',
                  fontSize: '12px',
                  whiteSpace: 'pre-wrap',
                  maxHeight: '400px',
                  overflow: 'auto',
                }}
              >
                <pre style={{ margin: 0 }}>{selectedSkill.content}</pre>
              </div>
            </div>

            {selectedSkill.bound_agents && selectedSkill.bound_agents.length > 0 && (
              <div style={{ marginTop: 24 }}>
                <Title level={5}>绑定的 Agent ({selectedSkill.bound_agents.length})</Title>
                <Space direction="vertical" style={{ width: '100%', marginTop: 12 }}>
                  {selectedSkill.bound_agents.map((agent) => (
                    <Card
                      key={agent.id}
                      size="small"
                      style={{ marginBottom: 8 }}
                    >
                      <Space>
                        <Tag color="blue">{agent.name}</Tag>
                        <span style={{ color: '#999', marginLeft: 8 }}>
                          {agent.agent_code}
                        </span>
                      </Space>
                      {agent.description && (
                        <div style={{ color: '#666', marginTop: 8, fontSize: '12px' }}>
                          {agent.description}
                        </div>
                      )}
                    </Card>
                  ))}
                </Space>
              </div>
            )}
          </div>
        ) : (
          <Empty description="技能不存在" />
        )}
      </Modal>
    </div>
  );
};

export default Skills;
