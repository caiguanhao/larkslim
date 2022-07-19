package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/caiguanhao/larkslim"
)

func die(a ...interface{}) {
	fmt.Fprintln(os.Stderr, a...)
	os.Exit(1)
}

func main() {
	var appId, appSecret, sendTarget string
	flag.StringVar(&appId, "app-id", "", "lark app id (you can also use env LARK_APP_ID)")
	flag.StringVar(&appSecret, "app-secret", "", "lark app secret (you can also use env LARK_APP_SECRET)")
	flag.StringVar(&sendTarget, "target", "", "send message to open_id, user_id, email or chat_id")
	flag.Usage = func() {
		fmt.Fprintf(
			flag.CommandLine.Output(),
			"Usage: %s [text ...] - %s\n", os.Args[0],
			"send text (or stdin) to lark",
		)
		flag.PrintDefaults()
	}
	flag.Parse()

	if appId == "" {
		appId = os.Getenv("LARK_APP_ID")
	}
	if appId == "" {
		die("error: empty app id")
	}

	if appSecret == "" {
		appSecret = os.Getenv("LARK_APP_SECRET")
	}
	if appSecret == "" {
		die("error: empty app secret")
	}

	var content string

	if len(flag.Args()) == 0 {
		fmt.Fprintln(os.Stderr, "Reading from stdin...")
		text, _ := io.ReadAll(os.Stdin)
		content = string(text)
	} else {
		content = strings.Join(flag.Args(), " ")
	}

	l := larkslim.API{
		AppId:     appId,
		AppSecret: appSecret,
	}

	err := l.SendMessage(sendTarget, content)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}