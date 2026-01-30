package models

import "encoding/json"

type QQMusicReq struct {
	Req0 struct {
		Module string `json:"module"`
		Method string `json:"method"`
		Param  struct {
			Disstid    int    `json:"disstid"`
			EncHostUin string `json:"enc_host_uin"`
			Tag        int    `json:"tag"`
			Userinfo   int    `json:"userinfo"`
			SongBegin  int    `json:"song_begin"`
			SongNum    int    `json:"song_num"`
		} `json:"param"`
	} `json:"req_0"`
	Comm struct {
		GTk      int    `json:"g_tk"`
		Uin      int    `json:"uin"`
		Format   string `json:"format"`
		Platform string `json:"platform"`
	} `json:"comm"`
}

func NewQQMusicReq(disstid int, platform string, songBegin, songNum int) *QQMusicReq {
	return &QQMusicReq{
		Req0: struct {
			Module string `json:"module"`
			Method string `json:"method"`
			Param  struct {
				Disstid    int    `json:"disstid"`
				EncHostUin string `json:"enc_host_uin"`
				Tag        int    `json:"tag"`
				Userinfo   int    `json:"userinfo"`
				SongBegin  int    `json:"song_begin"`
				SongNum    int    `json:"song_num"`
			} `json:"param"`
		}{
			Module: "music.srfDissInfo.aiDissInfo",
			Method: "uniform_get_Dissinfo",
			Param: struct {
				Disstid    int    `json:"disstid"`
				EncHostUin string `json:"enc_host_uin"`
				Tag        int    `json:"tag"`
				Userinfo   int    `json:"userinfo"`
				SongBegin  int    `json:"song_begin"`
				SongNum    int    `json:"song_num"`
			}{
				Disstid:    disstid,
				EncHostUin: "",
				Tag:        1,
				Userinfo:   1,
				SongBegin:  songBegin,
				SongNum:    songNum,
			},
		},
		Comm: struct {
			GTk      int    `json:"g_tk"`
			Uin      int    `json:"uin"`
			Format   string `json:"format"`
			Platform string `json:"platform"`
		}{
			GTk:      5381,
			Uin:      0,
			Format:   "json",
			Platform: platform,
		},
	}
}

// GetQQMusicReqStringWithPagination 获取带分页参数的请求字符串
func GetQQMusicReqStringWithPagination(disstid int, platform string, songBegin, songNum int) string {
	param := NewQQMusicReq(disstid, platform, songBegin, songNum)
	marshal, _ := json.Marshal(param)
	return string(marshal)
}

type QQMusicResp struct {
	Code int `json:"code"`
	Req0 struct {
		Code int `json:"code"`
		Data struct {
			Dirinfo struct {
				Title   string `json:"title"`
				Songnum int    `json:"songnum"`
			} `json:"dirinfo"`
			Songlist []struct {
				Name   string `json:"name"`
				Singer []struct {
					Name string `json:"name"`
				} `json:"singer"`
			} `json:"songlist"`
		} `json:"data"`
	} `json:"req_0"`
}

// QQMusicLegacyResp 旧版API响应结构（fcg_ucc_getcdinfo_byids_cp.fcg）
// 该接口能返回完整的歌曲列表，不受30首限制
type QQMusicLegacyResp struct {
	Code   int `json:"code"`
	Cdlist []struct {
		Dissname string `json:"dissname"`
		Songnum  int    `json:"songnum"`
		Songlist []struct {
			Songname string `json:"songname"`
			Singer   []struct {
				Name string `json:"name"`
			} `json:"singer"`
		} `json:"songlist"`
	} `json:"cdlist"`
}

// ToStandardResp 将旧版响应转换为标准响应格式
func (r *QQMusicLegacyResp) ToStandardResp() *QQMusicResp {
	if r.Code != 0 || len(r.Cdlist) == 0 {
		return nil
	}

	cd := r.Cdlist[0]
	resp := &QQMusicResp{
		Code: r.Code,
	}
	resp.Req0.Code = 0
	resp.Req0.Data.Dirinfo.Title = cd.Dissname
	resp.Req0.Data.Dirinfo.Songnum = cd.Songnum

	// 转换歌曲列表
	for _, song := range cd.Songlist {
		converted := struct {
			Name   string `json:"name"`
			Singer []struct {
				Name string `json:"name"`
			} `json:"singer"`
		}{
			Name: song.Songname,
		}
		for _, s := range song.Singer {
			converted.Singer = append(converted.Singer, struct {
				Name string `json:"name"`
			}{Name: s.Name})
		}
		resp.Req0.Data.Songlist = append(resp.Req0.Data.Songlist, converted)
	}

	return resp
}
