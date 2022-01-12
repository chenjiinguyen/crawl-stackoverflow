package main

import (
	"bufio"
	"math/rand"
	"encoding/json"
	"fmt"
	"io/ioutil"
	_ "net"
	"net/http"
	urls "net/url"
	"os"
	"strconv"
	"strings"
	"time"

	md "github.com/JohannesKaufmann/html-to-markdown"
	goquery "github.com/PuerkitoBio/goquery"
	freeproxy "github.com/soluchok/freeproxy"
	errgroup "golang.org/x/sync/errgroup"
	socks "h12.io/socks"
)

type Stack struct {
	Url      string    `json:"url"`
	Id       string    `json:"id"`
	Title    string    `json:"title"`
	Question string    `json:"question"`
	Answers  []string  `json:"answers"`
	Tags     []string  `json:"tags"`
	Created  time.Time `json:"created"`
}

type Data struct {
	index 	int
	socksProxy string
	links  []string `json:"links"`
	stacks []Stack  `json:"stacks"`
}

func HtmlToMarkDown(html string) string {
	converter := md.NewConverter("", true, nil)
	markdown, _ := converter.ConvertString(html)
	return markdown
}

func indexOf(element string, data []string) int {
	for k, v := range data {
		if element == v {
			return k
		}
	}
	return -1 //not found.
}

func RemoveIndex(s []string, index int) []string {
	return append(s[:index], s[index+1:]...)
}

func (data *Data) getStackByUrl(url string) (error, int) {
	req, _ := http.NewRequest("GET", url, nil)
	proxyUrl, _ := urls.Parse(data.socksProxy)
	_ = proxyUrl
	dialSocksProxy := socks.Dial(data.socksProxy)
	_ = dialSocksProxy
	req.Close = true
	i := &http.Transport{
		Proxy: http.ProxyURL(proxyUrl),
		// Dial: dialSocksProxy,
	}
	client := &http.Client{}
	client.Timeout = 10 * time.Second
	client.Transport = i
	resp, e := client.Do(req)
	if e != nil {
		return e, 500
	}
	if resp.StatusCode != 200 {
		if resp.StatusCode == 429 {
			fmt.Println("[",data.socksProxy,"] [", len(data.stacks), ",", len(data.links), "]", "Too many requests")
			return e , 429
		}
		return e, 500
	}
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return err, 500
	}
	defer resp.Body.Close()

	urlParts := strings.Split(url, "/")

	stackUrl := url
	stackID := urlParts[4]
	stackTitle := doc.Find("#question-header h1").Text()
	stackTags := []string{}
	stackAnswers := []string{}
	stackQuestionHtml, _ := doc.Find("#question .s-prose").Html()
	stackQuestion := HtmlToMarkDown(stackQuestionHtml)
	stackCreatedAt := time.Now()
	doc.Find("div.post-taglist a").Each(func(i int, s *goquery.Selection) {
		stackTags = append(stackTags, s.Text())
	})
	doc.Find("#answers .answer .js-post-body").EachWithBreak(func(i int, s *goquery.Selection) bool {
		if i == 3 {
			return false
		}
		html, _ := s.Html()
		markdown := HtmlToMarkDown(html)
		stackAnswers = append(stackAnswers, markdown)
		return true
	})

	doc.Find(".related .spacer").Each(func(i int, s *goquery.Selection) {
		relatedElement := s.Find("a")
		relatedUrl, _ := relatedElement.Attr("href")
		relatedId := strings.Split(relatedUrl, "/")[2]
		check := false
		for _, v := range data.stacks {
			if v.Id == relatedId {
				check = true
				break
			}
		}
		if check == false {
			data.links = append(data.links, "https://stackoverflow.com"+relatedUrl)
		}
	})

	stack := Stack{
		Url:      stackUrl,
		Id:       stackID,
		Title:    stackTitle,
		Question: stackQuestion,
		Answers:  stackAnswers,
		Tags:     stackTags,
		Created:  stackCreatedAt,
	}

	data.stacks = append(data.stacks, stack)

	index := indexOf(url, data.links)
	if index != -1 {
		data.links = RemoveIndex(data.links, index)
	}

	fmt.Println("[",data.index,"] [",data.socksProxy,"] [", len(data.stacks), ",", len(data.links), "]", "[", stackID, "]", stackTitle)
	return nil, 200
}

func (data *Data) getAllStacks(currentUrl string) {
	// eg := errgroup.Group{}
	// _ = eg
	// for len(data.links) > 0 {

	// 	for i := 0; i < len(data.links); i++ { // Lặp qua từng trang đã được phân trang
	// 		if i == 5 {
	// 			break
	// 		}
	// 		link := data.links[i]
	// 		eg.Go(func() error { // Tạo ra số lượng group goroutines bằng với số page, cùng đồng thời đi thu thập thông tin ebook
	// 			err := data.getStackByUrl(link) // Thu thập thông tin ebook qua url của page
	// 			if err != nil {
	// 				return err
	// 			}
	// 			return nil
	// 		})
	// 	}
	// 	result, _ := json.Marshal(data.stacks)
	// 	_ = ioutil.WriteFile("output.json", result, 0644)
	// }
	gen := freeproxy.New()
	data.socksProxy = "http://"+gen.Get()
	for i := 0; i < len(data.links); i++ { // Lặp qua từng trang đã được phân trang
		link := data.links[i]
		err, code := data.getStackByUrl(link)
		if err != nil || code == 429 {
			data.socksProxy = "http://"+gen.Get()
			i--
			continue
		}
		result, _ := json.Marshal(data.stacks)
		_ = ioutil.WriteFile("output/output"+strconv.Itoa(data.index)+".json", result, 0644)
		result_links, _ := json.Marshal(data.links)
		_ = ioutil.WriteFile("link/link"+strconv.Itoa(data.index)+".json", result_links, 0644)
	}
}

func readLine(path string) []string {
	inFile, err := os.Open(path)
	if err != nil {
	   fmt.Println(err.Error() + `: ` + path)
	   return []string{}
	}
	defer inFile.Close()
  
	scanner := bufio.NewScanner(inFile)
	result := []string{}
	for scanner.Scan() {
	  result = append(result,scanner.Text()) // the line
	}
	return result
}

func main() {
	urls := readLine("links.txt")
	eg := errgroup.Group{}
		for i := 1; i <= 12; i++ { // Lặp qua từng trang đã được phân trang
			eg.Go(func() error { // Tạo ra số lượng group goroutines bằng với số page, cùng đồng thời đi thu thập thông tin ebook
				data := Data{}
				data.index = rand.Intn(10000)
				url := urls[rand.Intn(len(urls))]
				data.links = append(data.links, url)
				data.getAllStacks(url) // Thu thập thông tin ebook qua url của page
				return nil
			})
		}
		if err := eg.Wait(); err != nil { // Error Group chờ đợi các group goroutines done, nếu có lỗi thì trả về
			fmt.Println(err)
		}
	
	

	// resp, err := client.Get("http://www.whatsmyip.us/")
	// fmt.Printf("%+v, %+v", resp, err)
}
