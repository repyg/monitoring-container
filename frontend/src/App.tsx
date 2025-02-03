import React, { useEffect, useState } from 'react';
import { Table, Card, Space, Tag, Typography } from 'antd';
import 'antd/dist/reset.css';

const { Title } = Typography;

interface PingResult {
  ip: string;
  ping_time: number;
  last_success: string;
  name: string;
  status: string;
  created: string;
}

const App: React.FC = () => {
  const [data, setData] = useState<PingResult[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const fetchData = async () => {
      try {
        setLoading(true);
        const response = await fetch(`${process.env.REACT_APP_API_URL}/api/ping-results`);
        const results = await response.json();
        setData(results);
      } catch (error) {
        console.error('Error fetching data:', error);
      } finally {
        setLoading(false);
      }
    };

    fetchData();
    const interval = setInterval(fetchData, 5000);
    return () => clearInterval(interval);
  }, []);

  const getStatusColor = (status: string): string => {
    if (status.includes('Up')) return 'green';
    if (status.includes('Exited')) return 'red';
    return 'orange';
  };

  const columns = [
    {
      title: 'Контейнер',
      dataIndex: 'name',
      key: 'name',
      render: (text: string) => text.replace('/', ''),
    },
    {
      title: 'IP Адрес',
      dataIndex: 'ip',
      key: 'ip',
    },
    {
      title: 'Статус',
      dataIndex: 'status',
      key: 'status',
      render: (status: string) => (
        <Tag color={getStatusColor(status)}>{status}</Tag>
      ),
    },
    {
      title: 'Время пинга (мс)',
      dataIndex: 'ping_time',
      key: 'ping_time',
      render: (time: number) => (
        <Tag color={time > 1000 ? 'red' : 'green'}>{time.toFixed(2)}</Tag>
      ),
    },
    {
      title: 'Последний успешный пинг',
      dataIndex: 'last_success',
      key: 'last_success',
      render: (date: string) => new Date(date).toLocaleString(),
    },
    {
      title: 'Создан',
      dataIndex: 'created',
      key: 'created',
      render: (date: string) => new Date(date).toLocaleString(),
    },
  ];

  return (
    <div style={{ padding: 24 }}>
      <Space direction="vertical" size="middle" style={{ width: '100%' }}>
        <Title level={2}>Мониторинг Docker-контейнеров</Title>
        <Card>
          <Table 
            dataSource={data} 
            columns={columns} 
            rowKey="ip"
            loading={loading}
            pagination={false}
          />
        </Card>
      </Space>
    </div>
  );
};

export default App; 