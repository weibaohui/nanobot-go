import React, { useEffect, useState } from 'react';
import {
  Table,
  Button,
  Space,
  Tag,
  Modal,
  Form,
  Input,
  Switch,
  message,
  Popconfirm,
  Card,
  DatePicker,
} from 'antd';
import { PlusOutlined, EditOutlined, DeleteOutlined, KeyOutlined, MessageOutlined } from '@ant-design/icons';
import { Tooltip } from 'antd';
import { usersApi, authApi, conversationsApi } from '../api';
import type { User, CreateUserRequest, ConversationRecord } from '../types';
import dayjs from 'dayjs';

const Users: React.FC = () => {
  const [users, setUsers] = useState<User[]>([]);
  const [loading, setLoading] = useState(false);
  const [modalVisible, setModalVisible] = useState(false);
  const [passwordModalVisible, setPasswordModalVisible] = useState(false);
  const [editingUser, setEditingUser] = useState<User | null>(null);
  const [selectedUser, setSelectedUser] = useState<User | null>(null);
  const [form] = Form.useForm();
  const [passwordForm] = Form.useForm();

  // 用户对话状态（默认昨天，因为今天还在发生中）
  const [conversationModalVisible, setConversationModalVisible] = useState(false);
  const [conversationLoading, setConversationLoading] = useState(false);
  const [conversationRecords, setConversationRecords] = useState<ConversationRecord[]>([]);
  const [selectedUserForConversation, setSelectedUserForConversation] = useState<User | null>(null);
  const [selectedDate, setSelectedDate] = useState(dayjs().subtract(1, 'day'));

  const fetchUsers = async () => {
    setLoading(true);
    try {
      const res = await usersApi.list();
      // usersApi.list 返回 ListResponse { items, total }
      setUsers((res as any)?.items || []);
    } catch (error) {
      message.error('获取用户列表失败');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchUsers();
  }, []);

  const handleCreate = async (values: CreateUserRequest) => {
    try {
      await usersApi.create(values);
      message.success('创建成功');
      setModalVisible(false);
      form.resetFields();
      fetchUsers();
    } catch (error) {
      message.error('创建失败');
    }
  };

  const handleUpdate = async (values: Partial<CreateUserRequest>) => {
    if (!editingUser) return;
    try {
      await usersApi.update(editingUser.id, values);
      message.success('更新成功');
      setModalVisible(false);
      setEditingUser(null);
      form.resetFields();
      fetchUsers();
    } catch (error) {
      message.error('更新失败');
    }
  };

  const handleDelete = async (id: number) => {
    try {
      await usersApi.delete(id);
      message.success('删除成功');
      fetchUsers();
    } catch (error) {
      message.error('删除失败');
    }
  };

  const handleChangePassword = async (values: { old_password: string; new_password: string }) => {
    if (!selectedUser) return;
    try {
      await authApi.changePassword(selectedUser.id, values.old_password, values.new_password);
      message.success('密码修改成功');
      setPasswordModalVisible(false);
      passwordForm.resetFields();
      setSelectedUser(null);
    } catch (error) {
      message.error('密码修改失败');
    }
  };

  const fetchUserConversations = async (user: User, date: dayjs.Dayjs) => {
    setConversationLoading(true);
    try {
      const dateStr = date.format('YYYY-MM-DD');
      const res = await conversationsApi.getByUserAndDate(user.user_code || user.username, dateStr);
      setConversationRecords(res as any);
    } catch (error) {
      message.error('获取用户对话失败');
    } finally {
      setConversationLoading(false);
    }
  };

  const columns = [
    { title: 'ID', dataIndex: 'id', width: 60 },
    { title: '用户编码', dataIndex: 'user_code', ellipsis: true },
    { title: '用户名', dataIndex: 'username' },
    { title: '邮箱', dataIndex: 'email' },
    { title: '显示名称', dataIndex: 'display_name' },
    {
      title: '状态',
      render: (_: any, record: User) => (
        <Tag color={record.is_active ? 'success' : 'default'}>
          {record.is_active ? '启用' : '禁用'}
        </Tag>
      ),
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      render: (date: string) => new Date(date).toLocaleString(),
    },
    {
      title: '操作',
      width: 150,
      render: (_: any, record: User) => (
        <Space size={4}>
          <Tooltip title="编辑">
            <Button
              type="text"
              icon={<EditOutlined />}
              onClick={() => {
                setEditingUser(record);
                form.setFieldsValue({
                  email: record.email,
                  display_name: record.display_name,
                  is_active: record.is_active,
                });
                setModalVisible(true);
              }}
            />
          </Tooltip>
          <Tooltip title="修改密码">
            <Button
              type="text"
              icon={<KeyOutlined />}
              onClick={() => {
                setSelectedUser(record);
                setPasswordModalVisible(true);
              }}
            />
          </Tooltip>
          <Tooltip title="查看对话">
            <Button
              type="text"
              icon={<MessageOutlined />}
              onClick={() => {
                const yesterday = dayjs().subtract(1, 'day');
                setSelectedUserForConversation(record);
                setSelectedDate(yesterday);
                fetchUserConversations(record, yesterday);
                setConversationModalVisible(true);
              }}
            />
          </Tooltip>
          <Popconfirm
            title="确认删除"
            description="删除后将无法恢复，是否继续？"
            onConfirm={() => handleDelete(record.id)}
          >
            <Tooltip title="删除">
              <Button type="text" danger icon={<DeleteOutlined />} />
            </Tooltip>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <div>
      <Card
        title="用户管理"
        extra={
          <Button
            type="primary"
            icon={<PlusOutlined />}
            onClick={() => {
              setEditingUser(null);
              form.resetFields();
              setModalVisible(true);
            }}
          >
            新建用户
          </Button>
        }
      >
        <Table rowKey="id" columns={columns} dataSource={users} loading={loading} />
      </Card>

      <Modal
        title={editingUser ? '编辑用户' : '新建用户'}
        open={modalVisible}
        onCancel={() => {
          setModalVisible(false);
          setEditingUser(null);
          form.resetFields();
        }}
        onOk={() => form.submit()}
      >
        <Form
          form={form}
          layout="vertical"
          onFinish={editingUser ? handleUpdate : handleCreate}
        >
          {!editingUser && (
            <Form.Item
              name="username"
              label="用户名"
              rules={[{ required: true, message: '请输入用户名' }]}
            >
              <Input placeholder="用户名" />
            </Form.Item>
          )}

          {!editingUser && (
            <Form.Item
              name="password"
              label="密码"
              rules={[{ required: true, message: '请输入密码' }]}
            >
              <Input.Password placeholder="密码" />
            </Form.Item>
          )}

          <Form.Item name="email" label="邮箱">
            <Input placeholder="邮箱" />
          </Form.Item>

          <Form.Item name="display_name" label="显示名称">
            <Input placeholder="显示名称" />
          </Form.Item>

          <Form.Item name="is_active" valuePropName="checked" initialValue={true}>
            <Switch checkedChildren="启用" unCheckedChildren="禁用" />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title="修改密码"
        open={passwordModalVisible}
        onCancel={() => {
          setPasswordModalVisible(false);
          setSelectedUser(null);
          passwordForm.resetFields();
        }}
        onOk={() => passwordForm.submit()}
      >
        <Form
          form={passwordForm}
          layout="vertical"
          onFinish={handleChangePassword}
        >
          <Form.Item
            name="old_password"
            label="旧密码"
            rules={[{ required: true, message: '请输入旧密码' }]}
          >
            <Input.Password placeholder="旧密码" />
          </Form.Item>

          <Form.Item
            name="new_password"
            label="新密码"
            rules={[{ required: true, message: '请输入新密码' }]}
          >
            <Input.Password placeholder="新密码" />
          </Form.Item>
        </Form>
      </Modal>

      {/* 用户对话记录弹窗 */}
      <Modal
        title={`${selectedUserForConversation?.display_name || selectedUserForConversation?.username || '用户'} 的对话记录`}
        open={conversationModalVisible}
        onCancel={() => {
          setConversationModalVisible(false);
          setConversationRecords([]);
          setSelectedUserForConversation(null);
        }}
        footer={
          <div style={{ display: 'flex', justifyContent: 'flex-end' }}>
            <Button onClick={() => setConversationModalVisible(false)}>关闭</Button>
          </div>
        }
        width={900}
      >
        <div style={{ marginBottom: 16 }}>
          <Space>
            <span>选择日期：</span>
            <DatePicker
              value={selectedDate}
              onChange={(date) => {
                if (date && selectedUserForConversation) {
                  setSelectedDate(date);
                  fetchUserConversations(selectedUserForConversation, date);
                }
              }}
              format="YYYY-MM-DD"
            />
            <Button
              onClick={() => {
                if (selectedUserForConversation) {
                  fetchUserConversations(selectedUserForConversation, selectedDate);
                }
              }}
            >
              刷新
            </Button>
          </Space>
        </div>

        {conversationLoading ? (
          <div style={{ textAlign: 'center', padding: 40 }}>加载中...</div>
        ) : conversationRecords.length === 0 ? (
          <div style={{ textAlign: 'center', padding: 40 }}>该日期暂无对话记录</div>
        ) : (
          <div style={{ maxHeight: 500, overflowY: 'auto', padding: 16, background: '#f5f5f5', borderRadius: 8 }}>
            {conversationRecords.map((record) => (
              <div
                key={record.id}
                style={{
                  display: 'flex',
                  flexDirection: record.role === 'user' ? 'row-reverse' : 'row',
                  marginBottom: 16,
                  alignItems: 'flex-start',
                }}
              >
                <div
                  style={{
                    maxWidth: '70%',
                    padding: '12px 16px',
                    borderRadius: record.role === 'user' ? '16px 16px 4px 16px' : '16px 16px 16px 4px',
                    background: record.role === 'user' ? '#1890ff' : '#fff',
                    color: record.role === 'user' ? '#fff' : '#333',
                    boxShadow: '0 2px 8px rgba(0,0,0,0.1)',
                  }}
                >
                  <div style={{ fontSize: 12, opacity: 0.7, marginBottom: 4 }}>
                    {record.role} · {record.timestamp ? new Date(record.timestamp).toLocaleTimeString() : '-'}
                  </div>
                  <div style={{ whiteSpace: 'pre-wrap', wordBreak: 'break-word' }}>
                    {record.content}
                  </div>
                </div>
              </div>
            ))}
          </div>
        )}
      </Modal>
    </div>
  );
};

export default Users;
