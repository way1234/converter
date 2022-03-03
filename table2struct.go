package converter

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	"github.com/way1234/converter/tool"
)

//map for converting mysql type to golang types
var typeForMysqlToGo = map[string]string{
	"int":                "int32",
	"integer":            "int32",
	"tinyint":            "int32",
	"smallint":           "int32",
	"mediumint":          "int32",
	"bigint":             "int64",
	"int unsigned":       "int32",
	"integer unsigned":   "int32",
	"tinyint unsigned":   "int32",
	"smallint unsigned":  "int32",
	"mediumint unsigned": "int32",
	"bigint unsigned":    "int64",
	"bit":                "int32",
	"bool":               "bool",
	"enum":               "string",
	"set":                "string",
	"varchar":            "string",
	"char":               "string",
	"tinytext":           "string",
	"mediumtext":         "string",
	"text":               "string",
	"longtext":           "string",
	"blob":               "string",
	"tinyblob":           "string",
	"mediumblob":         "string",
	"longblob":           "string",
	"date":               "time.Time", // time.Time or string
	"datetime":           "time.Time", // time.Time or string
	"timestamp":          "time.Time", // time.Time or string
	"time":               "time.Time", // time.Time or string
	"float":              "float64",
	"double":             "float64",
	"decimal":            "float64",
	"binary":             "string",
	"varbinary":          "string",
}

type Table2Struct struct {
	dsn                       string
	savePath                  string
	db                        *sql.DB
	table                     string
	prefix                    string
	config                    *T2tConfig
	err                       error
	realNameMethod            string
	enableJsonTag             bool   // 是否添加json的tag, 默认不添加
	packageName               string // 生成struct的包名(默认为空的话, 则取名为: package model)
	tagKey                    string // tag字段的key值,默认是orm
	jsonFieldToSmallCamelCase bool   // json字段采用小驼峰命名法
	tableNameToBigCamelCase   bool   // 表名采用大驼峰命名法
}

type T2tConfig struct {
	RmTagIfUcFirsted bool // 如果字段首字母本来就是大写, 就不添加tag, 默认false添加, true不添加
	TagToLower       bool // tag的字段名字是否转换为小写, 如果本身有大写字母的话, 默认false不转
	UcFirstOnly      bool // 字段首字母大写的同时, 是否要把其他字母转换为小写,默认false不转换
	SeperatFile      bool // 每个struct放入单独的文件,默认false,放入同一个文件
}

func NewTable2Struct() *Table2Struct {
	return &Table2Struct{}
}

func (t *Table2Struct) Dsn(d string) *Table2Struct {
	t.dsn = d
	return t
}

func (t *Table2Struct) TagKey(r string) *Table2Struct {
	t.tagKey = r
	return t
}

// 生成struct的包名(默认为空的话, 则取名为: package model)
func (t *Table2Struct) PackageName(r string) *Table2Struct {
	t.packageName = r
	return t
}

func (t *Table2Struct) RealNameMethod(r string) *Table2Struct {
	t.realNameMethod = r
	return t
}

func (t *Table2Struct) SavePath(p string) *Table2Struct {
	t.savePath = p
	return t
}

func (t *Table2Struct) DB(d *sql.DB) *Table2Struct {
	t.db = d
	return t
}

func (t *Table2Struct) Table(tab string) *Table2Struct {
	t.table = tab
	return t
}

func (t *Table2Struct) Prefix(p string) *Table2Struct {
	t.prefix = p
	return t
}

func (t *Table2Struct) EnableJsonTag(p bool) *Table2Struct {
	t.enableJsonTag = p
	return t
}

// json字段采用小驼峰命名法
func (t *Table2Struct) JsonFieldToSmallCamelCase(b bool) *Table2Struct {
	t.jsonFieldToSmallCamelCase = b
	return t
}

// 表名采用大驼峰命名法
func (t *Table2Struct) TableNameToBigCamelCase(b bool) *Table2Struct {
	t.tableNameToBigCamelCase = b
	return t
}

func (t *Table2Struct) Config(c *T2tConfig) *Table2Struct {
	t.config = c
	return t
}

func (t *Table2Struct) Run() error {
	if t.config == nil {
		t.config = new(T2tConfig)
	}
	// 连接mysql, 获取db对象
	t.dialMysql()
	if t.err != nil {
		return t.err
	}

	// 获取表和字段的shcema
	tableColumns, err := t.getColumns()
	if err != nil {
		return err
	}

	//fmt.Println(tableColumns)

	// 包名
	var packageName string
	if t.packageName == "" {
		packageName = "package model\n\n"
	} else {
		packageName = fmt.Sprintf("package %s\n\n", t.packageName)
	}

	// 组装struct
	var structContent string
	for tableRealName, item := range tableColumns {
		// 去除前缀
		if t.prefix != "" {
			tableRealName = tableRealName[len(t.prefix):]
		}
		tableName := tableRealName

		switch len(tableName) {
		case 0:
		case 1:
			tableName = strings.ToUpper(tableName[0:1])
		default:
			// 字符长度大于1时
			tableName = strings.ToUpper(tableName[0:1]) + tableName[1:]
		}

		if t.tableNameToBigCamelCase {
			tableName = tool.ToBigCamelCase(tableName)
		}

		depth := 1
		structContent += "// " + item.TableComment + "\n"
		structContent += "type " + tableName + " struct {\n"
		for _, v := range item.Columns {
			//structContent += tab(depth) + v.ColumnName + " " + v.Type + " " + v.Json + "\n"
			// 字段注释
			var clumnComment string
			if v.ColumnComment != "" {
				clumnComment = fmt.Sprintf(" // %s", v.ColumnComment)
			}
			structContent += fmt.Sprintf("%s%s %s %s%s\n",
				tab(depth), v.ColumnName, v.Type, v.Tag, clumnComment)
		}
		structContent += tab(depth-1) + "}\n\n"

		// 添加 method 获取真实表名
		if t.realNameMethod != "" {
			structContent += fmt.Sprintf("func (*%s) %s() string {\n",
				tableName, t.realNameMethod)
			structContent += fmt.Sprintf("%sreturn \"%s\"\n",
				tab(depth), tableRealName)
			structContent += "}\n\n"
		}
		fmt.Println(structContent)
	}

	// 如果有引入 time.Time, 则需要引入 time 包
	var importContent string
	if strings.Contains(structContent, "time.Time") {
		importContent = "import \"time\"\n\n"
	}

	// 写入文件struct
	var savePath = t.savePath
	// 是否指定保存路径
	if savePath == "" {
		savePath = "model.go"
	}
	filePath := savePath
	f, err := os.Create(filePath)
	if err != nil {
		fmt.Println("Can not write file")
		return err
	}
	defer f.Close()

	f.WriteString(packageName + importContent + structContent)

	cmd := exec.Command("gofmt", "-w", filePath)
	cmd.Run()

	return nil
}

func (t *Table2Struct) dialMysql() {
	if t.db == nil {
		if t.dsn == "" {
			t.err = errors.New("dsn数据库配置缺失")
			return
		}
		t.db, t.err = sql.Open("mysql", t.dsn)
	}
}

type column struct {
	ColumnName    string
	ColumnComment string
	Type          string
	Nullable      string
	TableName     string
	TableComment  string
	Tag           string
}
type table struct {
	TableName    string
	TableComment string
	Columns      []column
}

// Function for fetching schema definition of passed table
func (t *Table2Struct) getColumns() (tableColumns map[string]table, err error) {

	var sqlStr = `SELECT distinct c.COLUMN_NAME,c.DATA_TYPE,c.IS_NULLABLE,c.TABLE_NAME,c.COLUMN_COMMENT, t.TABLE_COMMENT
				FROM information_schema.COLUMNS c left join information_schema.Tables t on c.TABLE_NAME = t.TABLE_NAME
				WHERE c.table_schema = DATABASE()`

	// 是否指定了具体的table
	if t.table != "" {
		sqlStr += fmt.Sprintf(" AND c.TABLE_NAME = '%s'", (t.prefix + t.table))
	}

	// sql排序
	sqlStr += " order by c.TABLE_NAME asc, c.ORDINAL_POSITION asc"

	rows, err := t.db.Query(sqlStr)
	if err != nil {
		fmt.Println("Error reading table information: ", err.Error())
		return
	}

	defer rows.Close()

	tableColumns = make(map[string]table)
	for rows.Next() {
		col := column{}
		err = rows.Scan(&col.ColumnName, &col.Type, &col.Nullable, &col.TableName, &col.ColumnComment, &col.TableComment)

		if err != nil {
			fmt.Println(err.Error())
			return
		}

		col.Tag = col.ColumnName
		col.ColumnName = t.camelCase(col.ColumnName)
		col.Type = typeForMysqlToGo[col.Type]
		// 字段首字母本身大写, 是否需要删除tag
		if t.config.RmTagIfUcFirsted && col.ColumnName[0:1] == strings.ToUpper(col.ColumnName[0:1]) {
			col.Tag = "-"
		} else {
			// 是否需要将tag转换成小写
			if t.config.TagToLower {
				col.Tag = strings.ToLower(col.Tag)
			}
		}

		if t.tagKey == "" {
			t.tagKey = "orm"
		}

		if t.enableJsonTag {

			tag := col.Tag
			if t.jsonFieldToSmallCamelCase {
				tag = tool.ToSmallCamelCase(tag)
			}

			if col.Type == "int64" {
				tag += ",string"
			}

			col.Tag = fmt.Sprintf("`%s:\"%s\" json:\"%s\"`", t.tagKey, col.Tag, tag)
		} else {
			col.Tag = fmt.Sprintf("`%s:\"%s\"`", t.tagKey, col.Tag)
		}

		mt, ok := tableColumns[col.TableName]
		if !ok {
			mt = table{
				TableName:    col.TableName,
				TableComment: col.ColumnComment,
				Columns:      make([]column, 0),
			}
			//tableColumns[col.TableName] = mt
		}

		mt.Columns = append(mt.Columns, col)
		tableColumns[col.TableName] = mt
	}

	return
}

func (t *Table2Struct) camelCase(str string) string {

	// 是否有表前缀, 设置了就先去除表前缀
	if t.prefix != "" {
		str = strings.Replace(str, t.prefix, "", 1)
	}
	var text string
	//for _, p := range strings.Split(name, "_") {
	for _, p := range strings.Split(str, "_") {
		// 字段首字母大写的同时, 是否要把其他字母转换为小写
		switch len(p) {
		case 0:
		case 1:
			text += strings.ToUpper(p[0:1])
		default:
			// 字符长度大于1时
			if t.config.UcFirstOnly {
				text += strings.ToUpper(p[0:1]) + strings.ToLower(p[1:])
			} else {
				text += strings.ToUpper(p[0:1]) + p[1:]
			}
		}
	}
	return text
}

func tab(depth int) string {
	return strings.Repeat("\t", depth)
}
