package gosolver

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var Client = &http.Client{}
var Service string

func SolveHCaptcha(data SolverData) (Result, error) {
	url := ""
	task := ""

	Service := strings.ToLower(data.Service)

	switch Service {
	case "capsolver":
		url = CapsolverBaseURL + "/createTask"
	case "capmonster":
		url = CapmonsterBaseURL + "/createTask"
	case "2captcha":
		// implemented soon as well as more services perhaps.
	default:
		return Result{}, fmt.Errorf("unsupported service: %s", data.Service)
	}

	if data.Proxy == "" {
		task = "HCaptchaTaskProxyLess"
	} else {
		task = "HCaptchaTurboTask"
	}

	var payload interface{}

	switch Service {
	case "capsolver":
		taskPayload := TaskPayload{
			ClientKey: data.ClientKey,
			Task: HCaptchaTask{
				Type:        task,
				WebsiteURL:  data.WebURL,
				WebsiteKey:  data.SiteKey,
				IsInvisible: true,
				EnterprisePayload: struct {
					Rqdata string `json:"rqdata"`
				}{Rqdata: ""},
				Proxy:     "",
				UserAgent: "",
			},
		}

		if data.SiteKey == "" {
			return Result{}, errors.New("[GoSolver] - Please pass a website key with the context.")
		}

		if data.Proxy != "" {
			taskPayload.Task.Proxy = data.Proxy
		}

		if data.UA != "" {
			taskPayload.Task.UserAgent = data.UA
		}

		if data.RQData != "" {
			taskPayload.Task.EnterprisePayload.Rqdata = data.RQData
		}

		payload = &taskPayload

	case "capmonster":
		capmonsterPayload := Capmonster{
			Key: data.ClientKey,
			Task: CapmonsterTask{
				Type:       "HCaptchaTaskProxyless",
				WebsiteURL: data.WebURL,
				WebsiteKey: data.SiteKey,
			},
		}

		payload = &capmonsterPayload
	default:
		return Result{}, fmt.Errorf("unsupported service: %s", data.Service)
	}

	p, err := json.Marshal(payload)

	if err != nil {
		return Result{}, fmt.Errorf("[GoSolver] - Error while marshaling payload: %s", err)
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(p))

	if err != nil {
		return Result{}, fmt.Errorf("[GoSolver] - Failed to create new HTTP request: %s", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := Client.Do(req)

	if err != nil {
		return Result{}, fmt.Errorf("[GoSolver] - Failed to perform client request: %s", err)
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)

	if err != nil {
		return Result{}, fmt.Errorf("[GoSolver] - Failed to read response body: %s", err)
	}

	response, err := unmarshalResponse(body)

	if err != nil {
		return Result{}, fmt.Errorf("[GoSolver] - Failed to unmarshal response: %s", err)
	}

	if response.GetErrorID() != 0 {
		if err := ProcessError(body); err != nil {
			return Result{}, fmt.Errorf("[GoSolver] - Unknown error: %s", err)
		}
	}

	if taskID := response.GetTaskID(); taskID != "" && response.GetErrorID() == 0 {
		for {
			res, err := ProcessTask(TaskResult{
				Key: data.ClientKey,
			}, response.GetTaskID())

			if err != nil {
				return Result{}, err
			}

			switch res.CapStatus {
			case "ready":
				// Captcha has been solved successfully.
				fmt.Printf("[GoSolver] - Successfully solved captcha: %s\n", res.CapResp[:20]+"...")
				return Result{
					Captchakey:      res.CapKey,
					CaptchaResponse: res.CapResp,
				}, nil
			case "failed":
				return Result{}, fmt.Errorf("")
			}

			time.Sleep(5 * time.Second)
		}
	}

	return Result{}, nil
}

func unmarshalResponse(body []byte) (TaskResponse, error) {
	response := TaskResponse(nil)
	if Service == "capsolver" {
		var created CreatedCapS
		response = &created
	} else if Service == "capmonster" {
		var created CreatedCapM
		response = &created
	}

	if err := json.Unmarshal(body, response); err != nil {
		fmt.Printf("[GoSolver] - Failed to unmarshal event: %s\n", err)
		return nil, err
	}

	return response, nil
}

func ProcessTask(rq TaskResult, taskid interface{}) (Context, error) {
	url := ""

	switch Service {
	case "capsolver":
		url = CapsolverBaseURL + "/getTaskResult"
	case "capmonster":
		url = CapmonsterBaseURL + "/getTaskResult"
	}

	payload := struct {
		ClientKey string      `json:"clientKey"`
		TaskID    interface{} `json:"taskId"`
	}{
		ClientKey: rq.Key,
		TaskID:    taskid,
	}

	pl, err := json.Marshal(payload)

	if err != nil {
		return Context{}, err
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(pl))

	if err != nil {
		return Context{}, err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := Client.Do(req)

	if err != nil {
		return Context{}, err
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)

	if err != nil {
		return Context{}, err
	}

	if Service == "capsolver" {
		var r FinishedTaskCapSolver

		if err := json.Unmarshal(body, &r); err != nil {
			return Context{}, err
		}
		
		return Context{
			CapKey:    r.Solution.CaptchaKey,
			CapResp:   r.Solution.CaptchaResp,
			CapStatus: r.Status,
		}, nil

	} else if Service == "capmonster" {
		var r FinishedTaskCapMonster

		if err := json.Unmarshal(body, &r); err != nil {
			return Context{}, err
		}

		return Context{
			CapKey:    r.Sol.RespKey,
			CapResp:   r.Sol.GRecaptchaResponse,
			CapStatus: r.Status,
		}, nil
	}

	return Context{}, err
}

func ProcessError(resp []byte) error {
	var err Error
	var errorMessages map[string]string

	if err := json.Unmarshal(resp, &err); err != nil {
		fmt.Printf("Failed to unmarshal event: %s\n", err)
		return err
	}

	switch Service {
	case "capsolver":
		errorMessages = map[string]string{
			"ERROR_SERVICE_UNAVAILABLE": "Service is currently unavailable",
			"ERROR_RATE_LIMIT":          "Request frequency/rate limit exceeded",
			"ERROR_INVALID_TASK_DATA":   "Data parsing failure",
			"ERROR_BAD_REQUEST":         "Request error, please contact customer service if it persists",
			"ERROR_TASKID_INVALID":      "Task ID does not exist or is invalid",
			"ERROR_TASK_TIMEOUT":        "The task has timed out",
			"ERROR_SETTLEMENT_FAILED":   "Mission point settlement failure",
			"ERROR_KEY_DENIED_ACCESS":   "Wrong account key, please check your clientKey",
			"ERROR_ZERO_BALANCE":        "Insufficient account balance, please top up",
			"ERROR_TASK_NOT_SUPPORTED":  "Task types are not supported",
			"ERROR_CAPTCHA_UNSOLVABLE":  "Captcha not recognized, please try again",
			"ERROR_UNKNOWN_QUESTION":    "Wrong Question ID, the task cannot be processed",
			"ERROR_PROXY_BANNED":        "Proxy IP banned by target service",
			"ERROR_INVALID_IMAGE":       "Image size does not meet the requirements",
			"ERROR_PARSE_IMAGE_FAIL":    "Failed to parse image, please check the image BASE64 encoding",
			"ERROR_IP_BANNED":           "Your IP has been blocked for 10 minutes",
			"ERROR_KEY_TEMP_BLOCKED":    "Due to too many error requests, you have been blocked temporarily",
		}
	case "capmonster":
		errorMessages = map[string]string{
			"INVALID KEY":               "Account authorization key not found or has an incorrect format",
			"ZERO IMAGE SIZE":           "The size of the captcha you are uploading is less than 100 bytes",
			"BIG IMAGE SIZE":            "The size of the captcha you are uploading is more than 500,000 bytes",
			"NO FUNDS":                  "Account has zero balance. Add funds to continue recognition",
			"REQUEST IS NOT ALLOWED":    "Request with the current account key is not allowed from your IP. Open your account settings and add your IP to the trusted list",
			"CAPTCHA UNSOLVABLE":        "This type of captchas is not supported by the service or the image does not contain an answer. It could also mean that the image is corrupted or was incorrectly rendered",
			"CAPTCHA ID IS NOT FOUND":   "The captcha that you are requesting was not found. Make sure you are requesting a status update only within 5 minutes of uploading",
			"CAPTCHA IS NOT READY":      "The captcha has not yet been solved",
			"IP BANNED":                 "You have exceeded the limit of requests with the wrong api key, check the correctness of your api key in the control panel and after some time, try again",
			"INCORRECT METHOD":          "Incorrect captcha type (value of the «type» parameter)",
			"REQUEST LIMIT EXCEEDED":    "You have exceeded the limit of requests to receive an answer for one task. Try to request the result of the task no more than 1 time in 2 seconds",
			"THE DOMAIN IS NOT ALLOWED": "Captcha from some domains cannot be solved in CapMonster Cloud. If you try to create a task for such a domain, this error will return",
			"THE TOKEN IS EXPIRED":      "Captcha provider server reported that the additional token has expired. Try creating a task with a new token",
			"NO FREE SERVERS":           "At the moment there are no available servers for recognizing this task. Try again after a while",
			"INVALID RECAPTCHA SITEKEY": "Invalid sitekey",
			"INVALID RECAPTCHA DOMAIN":  "Invalid domain for sitekey",
			"RECAPTCHA TIMEOUT":         "The timeout of the ReCaptcha recognition has been exceeded, most likely due to a slow proxy or Google server",
			"YOUR IP IS BLOCKED":        "Your IP is not allowed to access this API due to a large number of errors",
			"FAILED TO CONNECT PROXY":   "Unable to connect to the proxy server, connection timeout",
			"THE PROXY IP IS BANNED":    "The proxy IP is banned in the target captcha service",
			"INCORRECT TASK TYPE":       "The task type is incorrect or not supported. Check the «type» property in the task object",
			"ERROR_TASK_ABSENT":         "Task object not found or invalid JSON was sent in createTask request",
		}
	default:
		errorMessages = map[string]string{
			"UNKNOWN ERROR": "Unknown error occurred",
		}
	}

	errorMessage, found := errorMessages[err.ErrorCode]
	if !found {
		errorMessage = "Unknown error occurred"
	}

	fmt.Printf("[GoSolver] - Failed to solve captcha: %s\n", errorMessage)

	return nil
}

func FormatProxyInput(proxy string) string {
	if !strings.Contains(proxy, "@") {
		fmt.Print("[GoSolver -> ProxyFormatter] - Proxy format: username:password@host:port")
		return ""
	}

	if !strings.Contains(proxy, "http://") {
		proxy = "http://" + proxy
	}

	pr, err := url.Parse(proxy)

	if err != nil {
		fmt.Printf("Failed to format proxy: %s\n", err)
		return ""
	}

	var Proxy string

	password, _ := pr.User.Password()

	if len(password) < len(pr.User.Username()) {
		Proxy = password + ":" + pr.User.Username()
	} else {
		Proxy = pr.User.Username() + ":" + password
	}

	Proxy += "@" + pr.Host

	if pr.Port() != "" {
		if !strings.Contains(pr.Host, ":"+pr.Port()) {
			Proxy += ":" + pr.Port()
		}
	}

	return "http://" + Proxy
}
