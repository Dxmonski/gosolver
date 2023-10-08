package gosolver

const (
	CapsolverBaseURL  = "https://api.capsolver.com"
	CapmonsterBaseURL = "https://api.capmonster.cloud"
	TwoCaptchaBaseURL = "https://api.2captcha.com/"
)

type SolverData struct {
	ClientKey string
	Service   string
	UA        string
	Proxy     string
	WebURL    string
	SiteKey   string
	RQData    string
	RQToken   string
}

type HCaptchaTask struct {
	Type              string `json:"type"`
	WebsiteURL        string `json:"websiteURL"`
	WebsiteKey        string `json:"websiteKey"`
	IsInvisible       bool   `json:"isInvisible"`
	EnterprisePayload struct {
		Rqdata string `json:"rqdata"`
	} `json:"enterprisePayload"`
	Proxy      string `json:"proxy"`
	GetCaptcha string `json:"getCaptcha"`
	UserAgent  string `json:"userAgent"`
}

type TaskPayload struct {
	ClientKey string       `json:"clientKey"`
	Task      HCaptchaTask `json:"task"`
}

type Created struct {
	ErrorID int    `json:"errorId"`
	TaskID  string `json:"taskId"`
	Status  string `json:"status"`
}

type Error struct {
	Error            int    `json:"errorId"`
	ErrorCode        string `json:"errorCode"`
	ErrorDescription string `json:"errorDescription"`
}

type TaskResult struct {
	Key    string
	TaskID string
}

type Finished struct {
	Err            int    `json:"errorId"`
	ErrCode        int    `json:"errorCode"`
	ErrDescription string `json:"errorDescription"`
	Solution       struct {
		CaptchaKey  string `json:"captchaKey"`
		CaptchaResp string `json:"gRecaptchaResponse"`
	}
	Status string `json:"status"`
}

type Context struct {
	CapKey    string
	CapResp   string
	CapStatus string
}

type Result struct {
	Captchakey      string
	CaptchaResponse string
}
