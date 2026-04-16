package api

import (
	"JJFreeBooks/crypto"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type ChapterDetail struct {
	Code            int    `json:"code"`
	Message         string `json:"message"`
	ChapterID       string `json:"chapterId"`
	ChapterName     string `json:"chapterName"`
	ChapterIntro    string `json:"chapterIntro"`
	ChapterSize     string `json:"chapterSize"`
	ChapterDate     string `json:"chapterDate"`
	SayBody         string `json:"sayBody"`
	UpDown          int    `json:"upDown"`
	Update          int    `json:"update"`
	Content         string `json:"content"`
	IsVip           int    `json:"isvip"`
	AuthorID        string `json:"authorid"`
	AutoBuyStatus   string `json:"autobuystatus"`
	NoteIsLock      int    `json:"noteislock"`
	SayBodyV2       string `json:"sayBodyV2"`
	ShowSayBodyPage string `json:"show_saybody_page"`
}

func (c *ChapterDetail) UnmarshalJSON(data []byte) error {
	type alias ChapterDetail
	aux := struct {
		*alias
		Code       json.RawMessage `json:"code"`
		UpDown     json.RawMessage `json:"upDown"`
		Update     json.RawMessage `json:"update"`
		IsVip      json.RawMessage `json:"isvip"`
		NoteIsLock json.RawMessage `json:"noteislock"`
	}{
		alias: (*alias)(c),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	var err error
	if c.Code, err = parseIntField(aux.Code, "code"); err != nil {
		return err
	}
	if c.UpDown, err = parseIntField(aux.UpDown, "upDown"); err != nil {
		return err
	}
	if c.Update, err = parseIntField(aux.Update, "update"); err != nil {
		return err
	}
	if c.IsVip, err = parseIntField(aux.IsVip, "isvip"); err != nil {
		return err
	}
	if c.NoteIsLock, err = parseIntField(aux.NoteIsLock, "noteislock"); err != nil {
		return err
	}

	return nil
}

func parseIntField(raw json.RawMessage, fieldName string) (int, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return 0, nil
	}

	var number int
	if err := json.Unmarshal(raw, &number); err == nil {
		return number, nil
	}

	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		if text == "" {
			return 0, nil
		}
		value, convErr := strconv.Atoi(text)
		if convErr != nil {
			return 0, fmt.Errorf("字段 %s 的值 %q 不是有效整数: %w", fieldName, text, convErr)
		}
		return value, nil
	}

	return 0, fmt.Errorf("字段 %s 不是有效整数: %s", fieldName, string(raw))
}

func GetChapterContent(novelId, chapterId string) (ChapterDetail, error) {
	appUrl := fmt.Sprintf("https://app-cdn.jjwxc.com/androidapi/chapterContent?novelId=%s&chapterId=%s", novelId, chapterId)
	res, err := http.Get(appUrl)
	if err != nil {
		return ChapterDetail{}, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			return
		}
	}(res.Body)
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return ChapterDetail{}, err
	}

	var result ChapterDetail
	err = json.Unmarshal(body, &result)
	if err != nil {
		return ChapterDetail{}, err
	}

	// 检查是否为VIP章节错误响应
	if result.Code == 1004 {
		// 如果有加密内容，尝试解密
		if result.Content != "" {
			// 尝试固定密钥解密（限免VIP章节通常用这种方式）
			decrypted, err := crypto.DesDecryptString(result.Content, "KW8Dvm2N", "1ae2c94b")
			if err == nil && decrypted != "" {
				result.Content = decrypted
				return result, nil
			} else {
				fmt.Printf("章节 %s 固定密钥解密失败\n", chapterId)
				result.Content = fmt.Sprintf("<VIP章节解密失败，原始内容: %s>", result.Content)
			}
		} else {
			result.Content = "<该章节为VIP章节，需要购买才能阅读>"
		}
	}

	// 处理正常响应或明文VIP内容
	if result.Content != "" {
		// 检查内容是否为加密内容（长度>30且不包含中文字符）
		if len(result.Content) > 30 && !ContainsChinese(result.Content) {
			decrypted, err := crypto.DesDecryptString(result.Content, "KW8Dvm2N", "1ae2c94b")
			if err == nil && decrypted != "" && ContainsChinese(decrypted) {
				result.Content = decrypted
			} else {
				fmt.Printf("章节 %s 解密失败，使用原始内容\n", chapterId)
			}
		}
	}

	return result, nil
}

func GetVIPChapterContent(token, novelId, chapterId string) (ChapterDetail, error) {
	timestamp := time.Now().UnixMilli()
	ciphertextStr := fmt.Sprintf("%s:%s:%s:%s", strconv.Itoa(int(timestamp)), token, novelId, chapterId)

	ciphertext, err := crypto.DesEncrypt([]byte(ciphertextStr), []byte("KW8Dvm2N"), []byte("1ae2c94b"))
	if err != nil {
		return ChapterDetail{}, err
	}
	escapedChannelBody := url.QueryEscape(ciphertext)

	appUrl := fmt.Sprintf("https://android.jjwxc.net/androidapi/chapterContent?readState=readahead&versionCode=454&sign=%s", escapedChannelBody)

	req, err := http.NewRequest("GET", appUrl, nil)
	if err != nil {
		return ChapterDetail{}, err
	}
	req.Header.Add("User-Agent", "Mozilla/5.0 (Linux; Android 15; PJX110 Build/UKQ1.231108.001; wv) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/139.0.7258.94 Mobile Safari/537.36/JINJIANG-Android/454(PJX110;Scale/2.55;isHarmonyOS/false)")
	req.Header.Add("Accept-Encoding", "")
	req.Header.Add("local_maxchapterid", "71")
	req.Header.Add("cacheshowed", "false")
	req.Header.Add("referer", "http://android.jjwxc.net/?v=454")
	req.Header.Add("not_tip", "readahead")
	req.Header.Add("versiontype", "reading")
	req.Header.Add("versiontype", "reading")
	req.Header.Add("source", "android")
	req.Header.Add("versioncode", "454")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return ChapterDetail{}, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			return
		}
	}(res.Body)
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return ChapterDetail{}, err
	}

	var result ChapterDetail
	accesskey := res.Header.Get("accesskey")
	keystring := res.Header.Get("keystring")

	// 检查是否需要动态解密
	responseText := string(body)
	isPay := !json.Valid(body) || !containsContent(responseText)

	if isPay && accesskey != "" && keystring != "" {
		// 使用动态解密
		decrypted, err := crypto.DynamicDecryptWithContent(responseText, accesskey, keystring)
		if err != nil {
			return ChapterDetail{}, fmt.Errorf("动态解密失败: %w", err)
		}

		// 尝试解析为JSON
		if err := json.Unmarshal([]byte(decrypted), &result); err == nil {
			// 检查是否需要进一步解密content字段 (基于JavaScript逻辑)
			if result.Content != "" && len(result.Content) > 30 {
				finalContent, err := crypto.DesDecryptString(result.Content, "KW8Dvm2N", "1ae2c94b")
				if err == nil && finalContent != "" {
					result.Content = finalContent
				} else {
					fmt.Printf("章节 %s 固定密钥解密失败，使用动态解密结果\n", chapterId)
				}
			}
			return result, nil
		}

		// 如果不是JSON，可能是纯文本，尝试直接解密
		if len(decrypted) > 30 {
			finalContent, err := crypto.DesDecryptString(decrypted, "KW8Dvm2N", "1ae2c94b")
			if err == nil && finalContent != "" {
				result.Content = finalContent
				return result, nil
			}
		}
		result.Content = decrypted
		return result, nil
	}

	// 处理普通响应或直接加密的content
	if json.Valid(body) {
		if err := json.Unmarshal(body, &result); err != nil {
			return ChapterDetail{}, err
		}

		// 检查content是否需要解密
		if result.Content != "" && len(result.Content) > 30 && !ContainsChinese(result.Content) {
			decrypted, err := crypto.DesDecryptString(result.Content, "KW8Dvm2N", "1ae2c94b")
			if err == nil && decrypted != "" && ContainsChinese(decrypted) {
				result.Content = decrypted
			} else {
				fmt.Printf("章节 %s 解密失败，使用原始内容\n", chapterId)
			}
		}
		return result, nil
	}

	return ChapterDetail{}, fmt.Errorf("无法处理响应格式")
}

// containsContent 检查JSON字符串是否包含content字段
func containsContent(jsonStr string) bool {
	var temp map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &temp); err != nil {
		return false
	}
	_, exists := temp["content"]
	return exists
}

// ContainsChinese 检查字符串是否包含中文字符
func ContainsChinese(text string) bool {
	for _, r := range text {
		if r >= 0x4e00 && r <= 0x9fa5 {
			return true
		}
	}
	return false
}
