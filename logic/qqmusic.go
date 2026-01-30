package logic

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"GoMusic/misc/httputil"
	"GoMusic/misc/log"
	"GoMusic/misc/models"
	"GoMusic/misc/utils"
)

// QQ音乐相关常量
const (
	// API相关
	qqMusicRedis  = "qq_music:%d"
	qqMusicAPIURL = "https://u6.y.qq.com/cgi-bin/musics.fcg?sign=%s&_=%d"

	// 旧版API（无30首歌曲限制，能返回完整歌单）
	qqMusicLegacyAPIURL = "http://c.y.qq.com/qzone/fcg-bin/fcg_ucc_getcdinfo_byids_cp.fcg"

	// 错误响应长度标识
	qqMusicErrorResponseLength = 108

	// 分页相关
	maxSongsPerPage = 1000  // 每页最大歌曲数
	maxTotalSongs   = 10000 // 最大支持的歌曲总数
)

// 链接类型正则表达式
var (
	// 短链接，需要重定向
	shortLinkRegex = regexp.MustCompile(`fcgi-bin`)

	// 详情页链接，包含details关键词
	detailsLinkRegex = regexp.MustCompile(`details`)

	// 包含id=数字的链接
	idParamLinkRegex = regexp.MustCompile(`id=\d+`)

	// 包含playlist/数字的链接
	playlistLinkRegex = regexp.MustCompile(`.*playlist/\d+$`)
)

// QQMusicDiscover 获取QQ音乐歌单信息
// link: 歌单链接
// detailed: 是否使用详细歌曲名（原始歌曲名，不去除括号等内容）
func QQMusicDiscover(link string, detailed bool) (*models.SongList, error) {
	// 1. 从链接中提取歌单ID
	tid, err := extractPlaylistID(link)
	if err != nil || tid == 0 {
		return nil, errors.New("无效的歌单链接")
	}

	// 2. 获取歌单数据
	responseData, err := fetchPlaylistData(tid)
	if err != nil {
		log.Errorf("获取QQ音乐歌单数据失败: %v", err)
		return nil, fmt.Errorf("获取歌单数据失败: %w", err)
	}

	// 3. 解析响应数据
	qqMusicResponse := &models.QQMusicResp{}
	if err = json.Unmarshal(responseData, qqMusicResponse); err != nil {
		log.Errorf("解析QQ音乐响应数据失败: %v", err)
		return nil, fmt.Errorf("解析歌单数据失败: %w", err)
	}

	// 4. 构建歌曲列表
	songList := buildSongList(qqMusicResponse, detailed)

	return songList, nil
}

// fetchPlaylistData 获取QQ音乐歌单数据
// 使用旧版API (fcg_ucc_getcdinfo_byids_cp.fcg) 获取完整歌单数据
// 该接口不受30首歌曲限制，能返回歌单中的所有歌曲
func fetchPlaylistData(tid int) ([]byte, error) {
	// 使用旧版API获取完整歌单
	data, err := fetchPlaylistViaLegacyAPI(tid)
	if err != nil {
		log.Warnf("旧版API获取歌单失败，尝试新版API: %v", err)
		// 回退到新版API（可能只有30首）
		return fetchPlaylistViaNewAPI(tid)
	}
	return data, nil
}

// fetchPlaylistViaLegacyAPI 使用旧版API获取歌单完整数据
// 接口: fcg_ucc_getcdinfo_byids_cp.fcg
// 优点: 能返回歌单中的所有歌曲，无数量限制
func fetchPlaylistViaLegacyAPI(tid int) ([]byte, error) {
	// 构建请求URL
	requestURL := fmt.Sprintf("%s?type=1&utf8=1&disstid=%d&loginUin=0", qqMusicLegacyAPIURL, tid)

	// 必要的请求头
	req, err := http.NewRequest("GET", requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("Referer", "https://y.qq.com/n/yqq/playlist")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	// 发送请求
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	// 旧接口返回的是 jsonCallback({...}) 格式，需要提取JSON部分
	dataStr := string(data)
	if strings.HasPrefix(dataStr, "jsonCallback(") && strings.HasSuffix(dataStr, ")") {
		dataStr = dataStr[len("jsonCallback(") : len(dataStr)-1]
		data = []byte(dataStr)
	}

	// 解析旧版响应
	legacyResp := &models.QQMusicLegacyResp{}
	if err = json.Unmarshal(data, legacyResp); err != nil {
		return nil, fmt.Errorf("解析旧版响应失败: %w", err)
	}

	// 检查响应是否有效
	if legacyResp.Code != 0 || len(legacyResp.Cdlist) == 0 {
		return nil, fmt.Errorf("旧版API返回无效响应: code=%d, cdlist长度=%d", legacyResp.Code, len(legacyResp.Cdlist))
	}

	// 转换为标准响应格式
	standardResp := legacyResp.ToStandardResp()
	if standardResp == nil {
		return nil, errors.New("转换响应格式失败")
	}

	// 转换为JSON
	result, err := json.Marshal(standardResp)
	if err != nil {
		return nil, fmt.Errorf("序列化响应失败: %w", err)
	}

	log.Infof("旧版API成功获取歌单，共%d首歌曲", len(legacyResp.Cdlist[0].Songlist))
	return result, nil
}

// fetchPlaylistViaNewAPI 使用新版API获取歌单数据（回退方案，可能只有30首）
func fetchPlaylistViaNewAPI(tid int) ([]byte, error) {
	return fetchPlaylistPage(tid, 0, maxSongsPerPage)
}

// fetchPlaylistBasicInfo 获取歌单基本信息（第一页数据）
func fetchPlaylistBasicInfo(tid int) ([]byte, error) {
	return fetchPlaylistPage(tid, 0, maxSongsPerPage)
}

// fetchPlaylistPage 获取歌单指定页的数据
func fetchPlaylistPage(tid int, songBegin, songNum int) ([]byte, error) {
	// 支持的平台列表
	platforms := []string{"-1", "android", "iphone", "h5", "wxfshare", "iphone_wx", "windows"}

	// QQ音乐API必须的请求头，缺少这些头会导致每次只返回30首歌曲
	headers := map[string]string{
		"Referer":    "https://y.qq.com/",
		"Origin":     "https://y.qq.com",
		"User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
	}

	var lastErr error
	var resp *http.Response

	// 尝试不同平台参数
	for _, platform := range platforms {
		// 1. 构建请求参数
		paramString := models.GetQQMusicReqStringWithPagination(tid, platform, songBegin, songNum)
		sign := utils.Encrypt(paramString)
		requestURL := fmt.Sprintf(qqMusicAPIURL, sign, time.Now().UnixMilli())

		// 2. 发送请求（使用带请求头的方法）
		resp, lastErr = httputil.PostWithHeaders(requestURL, strings.NewReader(paramString), headers)
		if lastErr != nil {
			log.Errorf("HTTP请求失败(平台:%s): %v", platform, lastErr)
			continue
		}

		// 3. 读取响应
		data, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		// 4. 检查响应是否有效
		// 108字节长度表示返回了错误信息，需要尝试其他平台
		if len(data) != qqMusicErrorResponseLength {
			return data, nil
		}
	}

	return nil, fmt.Errorf("尝试所有平台参数均失败: %w", lastErr)
}

// extractPlaylistID 从QQ音乐链接中提取歌单ID
func extractPlaylistID(link string) (int, error) {
	// 1. 处理playlist/数字格式的链接
	if playlistLinkRegex.MatchString(link) {
		return extractNumberAfterKeyword(link, "playlist/")
	}

	// 2. 处理id=数字格式的链接
	if idParamLinkRegex.MatchString(link) {
		return extractNumberAfterKeyword(link, "id=")
	}

	// 3. 处理需要重定向的短链接
	if shortLinkRegex.MatchString(link) {
		redirectedLink, err := httputil.GetRedirectLocation(link)
		if err != nil {
			log.Errorf("获取重定向链接失败: %v", err)
			return 0, fmt.Errorf("处理短链接失败: %w", err)
		}
		// 递归处理重定向后的链接
		return extractPlaylistID(redirectedLink)
	}

	// 4. 处理details页面链接
	if detailsLinkRegex.MatchString(link) {
		tidString, err := utils.GetQQMusicParam(link)
		if err != nil {
			log.Errorf("从details链接提取ID失败: %v", err)
			return 0, fmt.Errorf("提取歌单ID失败: %w", err)
		}

		tid, err := strconv.Atoi(tidString)
		if err != nil {
			log.Errorf("歌单ID转换为数字失败: %v", err)
			return 0, fmt.Errorf("歌单ID格式错误: %w", err)
		}

		return tid, nil
	}

	return 0, errors.New("无效的歌单链接格式")
}

// buildSongList 根据QQ音乐响应数据构建歌曲列表
func buildSongList(response *models.QQMusicResp, detailed bool) *models.SongList {
	songsCount := response.Req0.Data.Dirinfo.Songnum
	songList := response.Req0.Data.Songlist

	songs := make([]string, 0, len(songList))
	builder := strings.Builder{}

	for _, song := range songList {
		builder.Reset()

		// 根据detailed参数决定是否使用原始歌曲名
		if detailed {
			builder.WriteString(song.Name) // 使用原始歌曲名
		} else {
			builder.WriteString(utils.StandardSongName(song.Name)) // 去除多余符号
		}

		builder.WriteString(" - ")

		// 处理歌手信息
		singers := make([]string, 0, len(song.Singer))
		for _, singer := range song.Singer {
			singers = append(singers, singer.Name)
		}
		builder.WriteString(strings.Join(singers, " / "))

		songs = append(songs, builder.String())
	}

	return &models.SongList{
		Name:       response.Req0.Data.Dirinfo.Title,
		Songs:      songs,
		SongsCount: songsCount,
	}
}

// extractNumberAfterKeyword 从字符串中提取关键词后面的数字
func extractNumberAfterKeyword(s, keyword string) (int, error) {
	index := strings.Index(s, keyword)
	if index < 0 {
		return 0, fmt.Errorf("未找到关键词: %s", keyword)
	}

	// 提取关键词后面的所有数字
	startIndex := index + len(keyword)
	endIndex := len(s)

	// 找到数字结束的位置
	for i := startIndex; i < endIndex; i++ {
		if s[i] < '0' || s[i] > '9' {
			endIndex = i
			break
		}
	}

	// 提取并转换数字
	numStr := s[startIndex:endIndex]
	num, err := strconv.Atoi(numStr)
	if err != nil {
		return 0, fmt.Errorf("数字转换失败: %w", err)
	}

	return num, nil
}
