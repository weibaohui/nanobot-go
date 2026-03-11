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
} from 'antd';
import { PlusOutlined, EditOutlined, DeleteOutlined, KeyOutlined } from '@ant-design/icons';
import { usersApi } from '../api';
import type { User, CreateUserRequest } from '../types';

const Users: React.FC = () => {
  const [users, setUsers] = useState<User[]>([]);
  const [loading, setLoading] = useState(false);
  const [modalVisible, setModalVisible] = useState(false);
  const [passwordModalVisible, setPasswordModalVisible] = useState(false);
  const [editingUser, setEditingUser] = useState<User | null>(null);
  const [selectedUser, setSelectedUser] = useState<User | null>(null);
  const [form] = Form.useForm();
  const [passwordForm] = Form.useForm();

  const fetchUsers = async () => {
    setLoading(true);
    try {
      const res = await usersApi.list();
      setUsers(res.data?.items || []);
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
      await usersApi.changePassword(selectedUser.id, values.old_password, values.new_password);
      message.success('密码修改成功');
      setPasswordModalVisible(false);
      passwordForm.resetFields();
      setSelectedUser(null);
    } catch (error) {
      message.error('密码修改失败');
    }
  };

  const columns = [
    { title: 'ID', dataIndex: 'id', width: 60 },
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
      width: 250,
      render: (_: any, record: User) => (
        <Space>
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
          >
            编辑
          </Button>
          <Button
            type="text"
            icon={<KeyOutlined />}
            onClick={() => {
              setSelectedUser(record);
              setPasswordModalVisible(true);
            }}
          >
            修改密码
          </Button>
          <Popconfirm
            title="确认删除"
            description="删除后将无法恢复，是否继续？"
            onConfirm={() => handleDelete(record.id)}
          >
            <Button type="text" danger icon={<DeleteOutlined />}>
              删除
            </Button>
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
    </div>
  );
};

export default Users;
