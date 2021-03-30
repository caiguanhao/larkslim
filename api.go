package larkslim

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	Prefix = "https://open.feishu.cn/open-apis"
)

type (
	API struct {
		AppId     string
		AppSecret string

		Timeout time.Duration

		Debugger func(args ...interface{})

		accessToken       string
		accessTokenExpire int
	}

	Protected struct {
		Original interface{}
		Filtered interface{}
	}

	APIResponse struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}

	AccessTokenResponse struct {
		APIResponse
		Expire int    `json:"expire"`
		Token  string `json:"tenant_access_token"`
	}

	GroupResponse struct {
		APIResponse
		Data struct {
			ChatId string `json:"chat_id"`
		} `json:"data"`
	}

	GroupInfoResponse struct {
		APIResponse
		Data Group `json:"data"`
	}

	Group struct {
		Avatar      string `json:"avatar"`
		ChatId      string `json:"chat_id"`
		Description string `json:"description"`
		Name        string `json:"name"`
		OwnerOpenId string `json:"owner_open_id"`
		OwnerUserId string `json:"owner_user_id"`
		Members     []struct {
			OpenId string `json:"open_id"`
		} `json:"members"`
	}

	Groups []Group

	GroupsResponse struct {
		APIResponse
		Data struct {
			Groups Groups `json:"groups"`
		} `json:"data"`
	}

	MessageResponse struct {
		APIResponse
		Data struct {
			MessageId string `json:"message_id"`
		} `json:"data"`
	}

	UserInfo struct {
		Name   string `json:"name"`
		OpenId string `json:"open_id"`
	}

	UserInfos []UserInfo

	UserInfoResponse struct {
		APIResponse
		Data struct {
			UserInfos UserInfos `json:"user_infos"`
		} `json:"data"`
	}

	EventResponse struct {
		Type string `json:"type"`

		// type == "url_verification"
		Challenge string `json:"challenge"`

		// type == "event_callback"
		Event struct {
			ChatId           string `json:"open_chat_id"`
			Type             string `json:"type"`
			MsgType          string `json:"msg_type"`
			Text             string `json:"text"`
			TextWithoutAtBot string `json:"text_without_at_bot"`
			OpenId           string `json:"open_id"`
			UserOpenId       string `json:"user_open_id"`
		} `json:"event"`
	}

	UploadResponse struct {
		APIResponse
		Data struct {
			ImageKey string `json:"image_key"`
		} `json:"data"`
	}

	PostTag struct {
		Tag      string `json:"tag,omitempty"`
		Unescape bool   `json:"un_escape,omitempty"`
		Text     string `json:"text,omitempty"`
		Href     string `json:"href,omitempty"`
		UserId   string `json:"user_id,omitempty"`
		ImageKey string `json:"image_key,omitempty"`
		Width    int    `json:"width,omitempty"`
		Height   int    `json:"height,omitempty"`
	}

	PostLine []PostTag

	PostLines []PostLine

	PostOfLocale struct {
		Title   string    `json:"title"`
		Content PostLines `json:"content"`
	}

	Post map[string]PostOfLocale
)

func (api *API) newRequest(path string, reqBody interface{}) (req *http.Request, err error) {
	var body io.Reader
	var debug func()
	switch v := reqBody.(type) {
	case io.Reader:
		body = v
	case Protected:
		reqData, err := json.Marshal(v.Original)
		if err != nil {
			return nil, err
		}
		body = bytes.NewReader(reqData)
		debug = func() {
			reqDataFiltered, _ := json.Marshal(v.Filtered)
			api.Debugger("request body:", string(reqDataFiltered))
		}
	default:
		reqData, err := json.Marshal(v)
		if err != nil {
			return nil, err
		}
		body = bytes.NewReader(reqData)
		debug = func() {
			api.Debugger("request body:", string(reqData))
		}
	}
	if api.Debugger != nil && debug != nil {
		debug()
	}
	req, err = http.NewRequest("POST", Prefix+path, body)
	if err != nil {
		return
	}
	req.Header.Set("Authorization", "Bearer "+api.accessToken)
	return
}

func (api *API) do(req *http.Request, respData interface{}) (err error) {
	var resp *http.Response
	client := http.Client{
		Timeout: api.Timeout,
	}
	resp, err = client.Do(req)
	if err != nil {
		return
	}
	if api.Debugger != nil {
		api.Debugger(req.URL.String(), "->", resp.Status)
	}
	defer resp.Body.Close()
	var res []byte
	res, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}
	if api.Debugger != nil {
		api.Debugger("response body:", string(res))
	}
	var apiResp APIResponse
	err = json.Unmarshal(res, &apiResp)
	if err != nil {
		return
	}
	if apiResp.Msg != "ok" && apiResp.Msg != "success" {
		err = fmt.Errorf("not ok or success returned: %s", apiResp.Msg)
		return
	}
	if respData != nil {
		err = json.Unmarshal(res, respData)
	}
	return
}

func (api *API) NewRequest(path string, reqBody interface{}, respData interface{}) (err error) {
	var req *http.Request
	req, err = api.newRequest(path, reqBody)
	if err != nil {
		return
	}
	return api.do(req, respData)
}

func (api *API) GetAccessToken() (expire int, err error) {
	var data AccessTokenResponse
	err = api.NewRequest(
		// path
		"/auth/v3/tenant_access_token/internal/",

		// request body
		Protected{
			Original: map[string]string{
				"app_id":     api.AppId,
				"app_secret": api.AppSecret,
			},
			Filtered: map[string]string{
				"app_id":     api.AppId,
				"app_secret": "[filtered]",
			},
		},

		// response
		&data,
	)
	if err != nil {
		return
	}
	api.accessToken = data.Token
	api.accessTokenExpire = data.Expire
	expire = data.Expire
	return
}

func (api *API) ListAllChats() (groups Groups, err error) {
	var data GroupsResponse
	err = api.NewRequest(
		// path
		"/chat/v4/list/",

		// request body
		struct {
			PageSize string `json:"page_size"`
		}{"200"},

		// response
		&data,
	)
	groups = data.Data.Groups
	return
}

func (api *API) GetChatInfo(chatId string) (group Group, err error) {
	var data GroupInfoResponse
	err = api.NewRequest(
		// path
		"/chat/v4/info/",

		// request body
		struct {
			ChatId string `json:"chat_id"`
		}{chatId},

		// response
		&data,
	)
	if err != nil {
		return
	}
	group = data.Data
	return
}

func (api *API) GetUserInfo(userIds []string) (userInfos UserInfos, err error) {
	v := url.Values{}
	for _, userId := range userIds {
		v.Add("open_ids", userId)
	}
	var data UserInfoResponse
	err = api.NewRequest(
		// path
		"/contact/v1/user/batch_get",

		// request body
		v,

		// response
		&data,
	)
	if err != nil {
		return
	}
	userInfos = data.Data.UserInfos
	return
}

func (api *API) CreateChat(name, userOpenId string) (chatId string, err error) {
	var data GroupResponse
	err = api.NewRequest(
		// path
		"/chat/v4/create/",

		// request body
		struct {
			Name    string   `json:"name"`
			OpenIds []string `json:"open_ids"`
		}{name, []string{userOpenId}},

		// response
		&data,
	)
	if err != nil {
		return
	}
	chatId = data.Data.ChatId
	return
}

func (api *API) DestroyChat(chatId string) (err error) {
	err = api.NewRequest(
		// path
		"/chat/v4/disband/",

		// request body
		struct {
			ChatId string `json:"chat_id"`
		}{chatId},

		// response
		nil,
	)
	return
}

func (api *API) AddUsersToChat(chatId string, userIds []string) (err error) {
	var data GroupResponse
	err = api.NewRequest(
		// path
		"/chat/v4/chatter/add/",

		// request body
		struct {
			ChatId  string   `json:"chat_id"`
			OpenIDs []string `json:"open_ids"`
		}{chatId, userIds},

		// response
		&data,
	)
	if err != nil {
		return
	}
	return
}

func (api *API) RemoveUsersFromChat(chatId string, userIds []string) (err error) {
	err = api.NewRequest(
		// path
		"/chat/v4/chatter/delete/",

		// request body
		struct {
			ChatId  string   `json:"chat_id"`
			OpenIDs []string `json:"open_ids"`
		}{chatId, userIds},

		// response
		nil,
	)
	return
}

func (api *API) SendMessage(target, content string) (err error) {
	a, b, c, d := parseTarget(target)
	var data MessageResponse
	err = api.NewRequest(
		// path
		"/message/v4/send/",

		// request body
		struct {
			OpenId  *string     `json:"open_id,omitempty"`
			ChatId  *string     `json:"chat_id,omitempty"`
			Email   *string     `json:"email,omitempty"`
			UserId  *string     `json:"user_id,omitempty"`
			MsgType string      `json:"msg_type"`
			Content interface{} `json:"content"`
		}{a, b, c, d, "text", struct {
			Text string `json:"text"`
		}{content}},

		// response
		&data,
	)
	return
}

func (api *API) SendImageMessage(target, imageKey string) (err error) {
	a, b, c, d := parseTarget(target)
	var data MessageResponse
	err = api.NewRequest(
		// path
		"/message/v4/send/",

		// request body
		struct {
			OpenId  *string     `json:"open_id,omitempty"`
			ChatId  *string     `json:"chat_id,omitempty"`
			Email   *string     `json:"email,omitempty"`
			UserId  *string     `json:"user_id,omitempty"`
			MsgType string      `json:"msg_type"`
			Content interface{} `json:"content"`
		}{a, b, c, d, "image", struct {
			ImageKey string `json:"image_key"`
		}{imageKey}},

		// response
		&data,
	)
	return
}

func (api *API) SendPost(target string, post Post) (err error) {
	a, b, c, d := parseTarget(target)
	var data MessageResponse
	err = api.NewRequest(
		// path
		"/message/v4/send/",

		// request body
		struct {
			OpenId  *string     `json:"open_id,omitempty"`
			ChatId  *string     `json:"chat_id,omitempty"`
			Email   *string     `json:"email,omitempty"`
			UserId  *string     `json:"user_id,omitempty"`
			MsgType string      `json:"msg_type"`
			Content interface{} `json:"content"`
		}{a, b, c, d, "post", struct {
			Post Post `json:"post"`
		}{post}},

		// response
		&data,
	)
	return
}

func (api *API) UploadAvatarImage(file io.Reader) (key string, err error) {
	return api.uploadImage("avatar", file)
}

func (api *API) UploadMessageImage(file io.Reader) (key string, err error) {
	return api.uploadImage("message", file)
}

func (api *API) uploadImage(imageType string, file io.Reader) (key string, err error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	var part io.Writer
	part, err = writer.CreateFormFile("image", "image")
	if err != nil {
		return
	}
	_, err = io.Copy(part, file)
	if err != nil {
		return
	}
	writer.WriteField("image_type", imageType)
	err = writer.Close()
	if err != nil {
		return
	}
	var req *http.Request
	req, err = api.newRequest(
		// path
		"/image/v4/put/",

		// request body
		body,
	)
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	var data UploadResponse
	err = api.do(req, &data)
	if err == nil {
		key = data.Data.ImageKey
	}
	return
}

func (groups *Groups) String() string {
	if len(*groups) == 0 {
		return "no groups"
	}
	var b bytes.Buffer
	for i, group := range *groups {
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString(fmt.Sprintf("%d. ", i+1))
		b.WriteString(group.Name)
		b.WriteString(": ")
		b.WriteString(group.ChatId)
	}
	return b.String()
}

func (userInfos *UserInfos) String() string {
	if len(*userInfos) == 0 {
		return "no users"
	}
	var b bytes.Buffer
	for i, user := range *userInfos {
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString(fmt.Sprintf("%d. ", i+1))
		b.WriteString(user.Name)
		b.WriteString(": ")
		b.WriteString(user.OpenId)
	}
	return b.String()
}

func parseTarget(target string) (*string, *string, *string, *string) {
	if strings.HasPrefix(target, "ou_") {
		return &target, nil, nil, nil
	}
	if strings.HasPrefix(target, "oc_") {
		return nil, &target, nil, nil
	}
	if strings.HasPrefix(target, "@") {
		return nil, nil, &target, nil
	}
	return nil, nil, nil, &target
}
