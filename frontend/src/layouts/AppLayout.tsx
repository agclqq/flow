import { Layout, Menu } from 'antd'
import { Outlet, useLocation, useNavigate } from 'react-router-dom'

const { Header, Content } = Layout

const menuItems = [{ key: '/templates', label: '流水线模板' }]

export default function AppLayout() {
  const navigate = useNavigate()
  const location = useLocation()
  const selectedKey = location.pathname.startsWith('/templates') ? '/templates' : '/templates'

  return (
    <Layout style={{ minHeight: '100vh', width: '100%' }}>
      <Header style={{ display: 'flex', alignItems: 'center', gap: 24 }}>
        <div style={{ color: '#fff', fontWeight: 600 }}>Flow 流水线平台</div>
        <Menu
          theme="dark"
          mode="horizontal"
          selectedKeys={[selectedKey]}
          items={menuItems}
          onClick={({ key }) => navigate(key)}
          style={{ flex: 1, minWidth: 0 }}
        />
      </Header>
      <Content style={{ padding: 24, width: '100%', boxSizing: 'border-box' }}>
        <Outlet />
      </Content>
    </Layout>
  )
}
