package main

import (
	"fmt"

	"github.com/Dxmonski/gosolver"
)

func Solve() {
	result, err := gosolver.SolveHCaptcha(gosolver.SolverData{
		ClientKey: "",                   // Solver API key here
		Service:   "",                   // Solver name here: capsolver, capmonster
		UA:        "",                   // User Agent
		Proxy:     FormatProxyInput(""), // Proxy input user:pass@host:port if you switch up some of them, it will reformat them. [optional]
		WebURL:    "",                   // The target URL for example: https://discord.com/register [REQUIRED]
		SiteKey:   "",                   // Target SiteKey found when you get the captcha response from discord `captcha_sitekey` [REQUIRED]
		RQData:    "",                   // rq data is also found when you get the captcha response from discord `captcha_rqdata` [optional if the target has it then pass it]
		RQToken:   "",                   // rq token is also found when you get the captcha response from discord `captcha_rqtoken` [optional if the target has it then pass it]
	})

	if err != nil {
		fmt.Printf("Failed to solve captcha: %s\n", err)
	}

	fmt.Printf("Captcha Response: %s\n", result.CaptchaResponse)
	fmt.Printf("Captcha Key: %s\n", result.Captchakey)
}
