package entertainment

import sharedxlsx "github.com/Icingworld/doubao-space/internal/xlsx"

var entertainmentImportTemplateHeaders = []string{
	"作品类型",
	"作品名称",
	"作者或导演",
	"年份",
	"来源",
	"状态",
	"当前进度",
	"总进度",
	"评分",
	"个人评价",
	"备注",
	"收藏",
	"标签",
	"开始日期",
	"完成日期",
}

func entertainmentImportTemplateXLSX() ([]byte, error) {
	return sharedxlsx.CreateTemplate("娱乐导入", entertainmentImportTemplateHeaders)
}
