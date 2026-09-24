import { Table, Typography } from 'antd'
import { SYSTEM_PARAMS } from '../../constants/systemParams'

export default function SystemParamsTab() {
  return (
    <div>
      <Typography.Paragraph type="secondary">
        以下参数可在脚本与通知模板中引用，无需在 Tab1 中声明。输入 <code>{'{{'}</code> 可触发自动补全。
      </Typography.Paragraph>
      <Table
        rowKey="key"
        pagination={false}
        columns={[
          { title: '参数名', dataIndex: 'name' },
          { title: '引用示例', dataIndex: 'example' },
        ]}
        dataSource={SYSTEM_PARAMS}
      />
    </div>
  )
}
