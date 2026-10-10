import { Table, Typography } from 'antd'
import { SYSTEM_PARAMS } from '../../constants/systemParams'

export default function SystemParamsTab() {
  return (
    <div>
      <Typography.Paragraph type="secondary">
        以下参数可在脚本与通知模板中引用，无需在配置页声明。脚本编辑器中输入 <code>{'{{'}</code> 或{' '}
        <code>$</code> 可触发自动补全；运行时也可直接使用 shell 环境变量（<code>$KEY</code>）。
      </Typography.Paragraph>
      <Table
        rowKey="key"
        pagination={false}
        columns={[
          { title: '参数名', dataIndex: 'name', width: 160 },
          {
            title: '引用示例',
            width: 440,
            render: (_, row) => (
              <span style={{ fontFamily: 'monospace', fontSize: 13, wordBreak: 'break-all' }}>
                {row.example}
                <Typography.Text type="secondary"> / </Typography.Text>
                {row.shellExample}
              </span>
            ),
          },
          { title: '说明', dataIndex: 'description' },
        ]}
        dataSource={SYSTEM_PARAMS}
      />
    </div>
  )
}
