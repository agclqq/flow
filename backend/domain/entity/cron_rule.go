package entity

// PublishedCronRule 表示某模板最新已发布快照上的定时触发规则。
type PublishedCronRule struct {
	TemplateID int64
	VersionID  int64
	CronExpr   string
}
