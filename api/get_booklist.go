package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// BookListRoot 是整个 JSON 响应的根结构
type BookListRoot struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Data    Data   `json:"data"`
}

// Data 包含主要的业务数据
type Data struct {
	ChannelID       string      `json:"channelid"`
	NaturalRankID   int         `json:"natural_rank_id"`
	Data            []NovelData `json:"data"`
	ChannelRule     string      `json:"channelRule"`
	ChannelRuleDown string      `json:"channelRuleDown"`
}

// NovelData 代表每本小说的数据
type NovelData struct {
	NovelID         string `json:"novelId"`         // 小说的唯一标识符
	NovelName       string `json:"novelName"`       // 小说标题
	AuthorID        string `json:"authorId"`        // 作者的唯一标识
	AuthorName      string `json:"authorName"`      // 作者名字
	Cover           string `json:"cover"`           // 小说封面图片链接
	Local           string `json:"local"`
	LocalImg        string `json:"localImg"`
	NovelIntroshort string `json:"novelIntroshort"` // XwX疑似晋江的石山嗷QAQ
	NovelIntroShort string `json:"novelIntroShort"` // 一句话简介
	NovelIntro      string `json:"novelIntro"`      // 简介
	NovelStep       string `json:"novelStep"`       // 小说状态
	Tags            string `json:"tags"`            // 小说标签
	FreeDate        string `json:"freeDate"`        // 免费日期
	NowFree         string `json:"nowFree"`         // 当前是否免费
	NovelSize       string `json:"novelSize"`       // 小说文字总量
	NovelSizeFormat string `json:"novelSizeformat"` // 格式化的文件大小
	NovelClass      string `json:"novelClass"`      // 小说分类
	IsVipMonth      string `json:"isVipMonth"`      // 是否为包月会员
	RecommendInfo   string `json:"recommendInfo"`   // 推荐信息，本身是一个 JSON 字符串
}

func GetBooksList() (BookListRoot, error) {
	now := time.Now()
	date := now.Format("2006-01-02")

	channelBody := fmt.Sprintf(`{"date_free_%s":{"offset":"0","limit":"40"}}`, date)

	escapedChannelBody := url.QueryEscape(channelBody)

	appUrl := fmt.Sprintf("https://app-cdn.jjwxc.com/bookstore/getFullPageV1?channelBody=%s&channelMore=1", escapedChannelBody)

	res, err := http.Get(appUrl)
	if err != nil {
		return BookListRoot{}, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			return
		}
	}(res.Body)
	var result BookListRoot
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return BookListRoot{}, err
	}
	err = json.Unmarshal(body, &result)
	if err != nil {
		return BookListRoot{}, err
	}
	return result, nil
}
