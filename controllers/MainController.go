package controllers

import (
	"g8ink/models"
	"g8ink/tools"
	"strconv"
	"time"

	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/core/validation"
	beego "github.com/beego/beego/v2/server/web"
)

type MainController struct {
	beego.Controller
}

func (c *MainController) Home() {
	c.Layout = "layout.html"
	code := c.Ctx.Input.Param(":code")
	o := orm.NewOrm()
	c.Data["WEB_BACKGROUND"], _ = beego.AppConfig.String("WEB_BACKGROUND")         //网页背景
	c.Data["gnum"], _ = o.QueryTable("url").Count()                                //全站生成数量
	c.Data["unum"], _ = o.QueryTable("url").Filter("ip", c.Ctx.Input.IP()).Count() //根据用户ip查找生成数量
	c.Data["year"] = time.Now().Year()

	if code == "" {
		// 首页
		c.TplName = "index.html"
		return
	}

	//如果shortcode不等于空则在数据库里找是否存在
	url := models.Url{}
	o.QueryTable("url").Filter("ShortCode", code).One(&url)
	// 判断是否为链接是就跳转
	if len(url.OriginalUrl) > 7 && (url.OriginalUrl[0:7] == "http://" || url.OriginalUrl[0:8] == "https://") {
		// 跳转
		c.Redirect(url.OriginalUrl, 301)
		return
	}
	// 如果原url内容为空则跳转首页
	if url.OriginalUrl == "" {
		c.Redirect("/", 301)
		return
	}
	c.Data["data"] = url.OriginalUrl
	c.TplName = "nourl.html"

}

func (c *MainController) Generate() {
	// 定义返回数据的变量
	re := make(map[string]interface{})
	c.Data["json"] = &re

	// 获取设定
	HOST, _ := beego.AppConfig.String("HOST")
	MAX_URL, _ := beego.AppConfig.Int("MAX_URL")
	MAX_SHORTCODE, _ := beego.AppConfig.Int("MAX_SHORTCODE")
	MIN_SHORTCODE, _ := beego.AppConfig.Int("MIN_SHORTCODE")
	RAND_SHORTCODE, _ := beego.AppConfig.Int("RAND_SHORTCODE")

	//获取表单数据
	shortcode := c.GetString("code")
	originalurl := c.GetString("url")

	valid := validation.Validation{}

	if shortcode != "" {
		valid.AlphaNumeric(shortcode, "code").Message("自定义后缀只能为英文字符或数字")
		valid.MinSize(shortcode, MIN_SHORTCODE, "code").Message("自定义后缀的最小长度为" + strconv.Itoa(MIN_SHORTCODE))
		valid.MaxSize(shortcode, MAX_SHORTCODE, "code").Message("自定义后缀的最大长度为" + strconv.Itoa(MAX_SHORTCODE))
	}
	valid.Required(originalurl, "url").Message("url不能为空")
	valid.MaxSize(originalurl, MAX_URL, "url").Message("url最大长度为" + strconv.Itoa(MAX_URL))

	if valid.HasErrors() {
		re["Code"] = -1
		re["Message"] = "参数错误，" + valid.Errors[0].String()
		c.ServeJSON()
		return
	}

	//判断是否为封禁的ip
	if tools.Isbanip(c.Ctx.Input.IP()) {
		re["Code"] = -2
		re["Message"] = "你已被封禁"
		c.ServeJSON()
		return
	}

	//判断是否为封禁的host
	if tools.Isbanhost(originalurl) {
		re["Code"] = -2
		re["Message"] = "封禁的域名"
		c.ServeJSON()
		return
	}

	// 判断是否已经生成该url
	existshortcode := tools.Urlexist(originalurl)
	if existshortcode != "" {
		re["Code"] = 200
		re["Shorturl"] = HOST + "/" + existshortcode
		re["Message"] = "成功"
		c.ServeJSON()
		return
	}

	//生成code
	if shortcode == "" {
		shortcode = tools.Getshortcode(RAND_SHORTCODE)
	} else {
		//判断短代码是否存在
		if tools.Codeexist(shortcode) {
			re["Code"] = -1
			re["Message"] = "该短链接已存在"
			c.ServeJSON()
			return
		}

	}

	//插入数据
	err := models.UrlInsert(shortcode, originalurl, c.Ctx.Input.IP())

	if err != nil {
		re["Code"] = -1
		re["Message"] = "生成错误"
	} else {
		re["Code"] = 200
		re["Shorturl"] = HOST + "/" + shortcode
		re["Message"] = "成功"
	}

	c.ServeJSON()
}

func (c *MainController) Robots() {
	rebotstxt := `# robots.txt
User-agent: *
Disallow: 
	`
	c.Ctx.WriteString(rebotstxt)
}
