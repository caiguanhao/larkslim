package lark_test

import (
	"encoding/json"
	"os"

	"github.com/caiguanhao/lark-slim"
)

func ExamplePost() {
	post := lark.Post{
		"zh_cn": lark.PostOfLocale{
			Title: "post",
			Content: lark.PostLines{
				{
					{
						Tag:  "text",
						Text: "Name: ",
					},
					{
						Tag:  "a",
						Text: "Hello",
						Href: "https://www.google.com",
					},
				},
			},
		},
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "\t")
	enc.Encode(post)
	// Output:
	// {
	// 	"zh_cn": {
	// 		"title": "post",
	// 		"content": [
	// 			[
	// 				{
	// 					"tag": "text",
	// 					"text": "Name: "
	// 				},
	// 				{
	// 					"tag": "a",
	// 					"text": "Hello",
	// 					"href": "https://www.google.com"
	// 				}
	// 			]
	// 		]
	// 	}
	// }
}
