package task

import (
	"fmt"
	"github.com/polite007/Milkyway/config"
	"github.com/polite007/Milkyway/internal/utils/finger"
	"github.com/polite007/Milkyway/internal/utils/httpx"
	"github.com/polite007/Milkyway/pkg/color"
	"github.com/polite007/Milkyway/pkg/logger"
	"strings"
)

func newDirScanTask(targetList []string, dirList []string) ([]*config.Resps, error) {
	NewPool := NewWorkPool(config.Get().WorkPoolNum)
	NewPool.Start()

	type HostPath struct {
		host string
		path string
	}

	f := func(args any) (any, error) {
		p, ok := args.(HostPath)
		if !ok {
			return nil, config.GetErrors().ErrAssertion
		}
		if p.host[len(p.host)-1] == '/' {
			p.host = p.host[:len(p.host)-2]
		}
		isAlive, err := httpx.Get(p.host, nil, p.path)
		if err == nil && isAlive.StatusCode == 200 {
			return httpx.HandleResponse(isAlive)
		}
		return nil, config.GetErrors().ErrTaskFailed
	}

	go func() {
		for _, dir := range dirList {
			if dir == "" {
				continue
			}
			for _, targetUrl := range targetList {
				hostPath := HostPath{
					host: strings.TrimRight(targetUrl, "/"),
					path: "/" + strings.Trim(strings.Trim(dir, "/"), "\r"),
				}
				NewPool.Wg.Add(1)
				NewPool.TaskQueue <- newTask(hostPath, f)
			}
		}
		close(NewPool.TaskQueue) // 关闭任务队列
		NewPool.Wg.Wait()        // 等待消费者执行完全部任务
		close(NewPool.Result)    // 关闭结果队列
	}()

	var result []*config.Resps
	for res := range NewPool.Result {
		if res == nil {
			continue
		}
		resultSimple := res.(*config.Resps)
		if len(resultSimple.Body) < 25 {
			continue
		}
		var logOut string
		resultSimple.Cms, resultSimple.Tags = finger.WebFinger(resultSimple)
		if resultSimple.Cms == "" {
			logOut = fmt.Sprintf("[%s] %-25v len:%d title:%s header: %s",
				color.Green(resultSimple.StatusCode),
				resultSimple.Url,
				len(resultSimple.Body),
				color.Green(resultSimple.Title),
				color.Green(resultSimple.Server),
			)
		} else {
			logOut = fmt.Sprintf("[%s] %-25v len:%d title:%s header: %s cms: %s",
				color.Green(resultSimple.StatusCode),
				resultSimple.Url,
				len(resultSimple.Body),
				color.Green(resultSimple.Title),
				color.Green(resultSimple.Server),
				color.Red(resultSimple.Cms),
			)
		}
		logger.OutLog(logOut)
		result = append(result, resultSimple)
	}
	return result, nil
}
