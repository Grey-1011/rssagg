package main

import (
	"context"
	"encoding/xml"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/Grey-1011/rssagg/internal/database"
)

// scraper.go 定义了一个用于定期抓取RSS feeds的scraper。
// 它会定期从数据库中获取需要抓取的feeds，然后并发地处理这些feeds

// 这个函数是整个抓取过程的入口。它通过定时器定期执行抓取操作。
func startScraping(db *database.Queries, concurrency int, timeBetweenRequest time.Duration) {
	log.Printf("Collecting feeds every %s on %v goroutines...", timeBetweenRequest, concurrency)
	/*
	   1. 定时器ticker：
	   	使用time.NewTicker(timeBetweenRequest)创建一个定时器，
	   	该定时器每隔timeBetweenRequest时间触发一次。
	*/
	ticker := time.NewTicker(timeBetweenRequest)

	/*
		2. 获取feeds：
			在每次定时器触发时，调用数据库查询GetNextFeedsToFetch获取下一个要抓取的feeds列表。
	*/
	for ; ; <-ticker.C {
		feeds, err := db.GetNextFeedsToFetch(context.Background(), int32(concurrency))
		if err != nil {
			log.Println("Couldn't get next feeds to fetch", err)
			continue
		}
		log.Printf("Found %v feeds to fetch!", len(feeds))
		/*
		   3. 并发处理：
		   	为每个feed创建一个新的goroutine，
		   	并使用sync.WaitGroup等待所有goroutine完成。
		*/
		wg := &sync.WaitGroup{}
		for _, feed := range feeds {
			wg.Add(1)
			go scrapeFeed(db, wg, feed)
		}
		wg.Wait()
	}
}

// 这个函数负责处理单个feed的抓取和处理。
func scrapeFeed(db *database.Queries, wg *sync.WaitGroup, feed database.Feed) {
	defer wg.Done()
	/*
		1. 标记feed已抓取：调用MarkFeedFetched更新数据库，标记该feed已被抓取。
	*/
	_, err := db.MarkFeedFetched(context.Background(), feed.ID)
	if err != nil {
		log.Printf("Couldn't mark feed %s fetched: %v", feed.Name, err)
		return
	}
	/*
	   2. 抓取feed内容：调用fetchFeed函数抓取feed的内容，如果失败则记录错误并返回。
	*/
	feedData, err := fetchFeed(feed.Url)
	if err != nil {
		log.Printf("Couldn't collect feed %s: %v", feed.Name, err)
		return
	}
	/*
		3. 处理feed数据：遍历抓取到的feed中的每个item，并记录其标题。
	*/
	for _, item := range feedData.Channel.Item {
		log.Println("Found post", item.Title)
	}
	/*
		4. 记录结果：记录该feed中找到的所有posts的数量。
	*/
	log.Printf("Feed %s collected, %v posts found", feed.Name, len(feedData.Channel.Item))
}

type RSSFeed struct {
	Channel struct {
		Title       string    `xml:"title"`
		Link        string    `xml:"link"`
		Description string    `xml:"description"`
		Language    string    `xml:"language"`
		Item        []RSSItem `xml:"item"`
	} `xml:"channel"`
}

type RSSItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
}

/*
fetchFeed 函数从指定的 URL 抓取 RSS Feed 数据，解析为 RSSFeed 结构体并返回。
通过这种方式，你可以轻松地从 RSS Feed 获取和解析数据，以便在你的应用程序中使用。
*/
func fetchFeed(feedURL string) (*RSSFeed, error) {
	// 创建一个 http.Client，设置超时时间为10秒。
	httpClient := http.Client{
		Timeout: 10 * time.Second,
	}
	// 使用 httpClient.Get(feedURL) 发起 HTTP GET 请求。如果请求失败，返回错误。
	resp, err := httpClient.Get(feedURL)
	if err != nil {
		return nil, err
	}
	// defer resp.Body.Close() 确保在函数结束时关闭响应体。
	defer resp.Body.Close()

	// 使用 io.ReadAll(resp.Body) 读取响应体数据。如果读取失败，返回错误。
	dat, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var rssFeed RSSFeed
	/* 使用 xml.Unmarshal(dat, &rssFeed) 将读取到的数据解析为 rssFeed 结构体。
	如果解析失败，返回错误。
	*/
	err = xml.Unmarshal(dat, &rssFeed)
	if err != nil {
		return nil, err
	}

	return &rssFeed, nil
}
