package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
	"github.com/gomodule/redigo/redis"
	"time"
)
var ffmp = os.Args[3]
var pool = newPool()
func main() {
	client := pool.Get()
	defer client.Close()
	ans, err := client.Do("EXISTS", os.Args[1])
	if err != nil {
		panic(err)
	}
	lev:=fmt.Sprintf("%d",ans)
	if lev=="1" {
		start := time.Now()

		videoDir := "./videos"
		textDir := "./texts"
		audioDir := "./audios"

		videoFiles, err := getFiles(videoDir, ".mp4")
		if err != nil {
			log.Fatal(err)
		}
		videoFilesb, err := getFiles(videoDir, ".MP4")
		if err != nil {
			log.Fatal(err)
		}
		videoFiles=append(videoFiles, videoFilesb...)
		textFiles, err := getFiles(textDir, ".txt")
		if err != nil {
			log.Fatal(err)
		}
		audioFiles, err := getFiles(audioDir, ".mp3")
		if err != nil {
			log.Fatal(err)
		}
		audioFilesb, err := getFiles(audioDir, ".MP3")
		if err != nil {
			log.Fatal(err)
		}
		audioFiles=append(audioFiles, audioFilesb...)

		procs, _ :=strconv.Atoi(os.Args[2])
		videoCount := len(videoFiles)
		mode := os.Args[4]
		m,_:=strconv.Atoi(mode)
		if m == 1 {
			var wg sync.WaitGroup
			for videoCount-procs >= 0 {
				videoOtr:=videoFiles[videoCount-procs:videoCount]
				wg.Add(1)
				crVideos(videoOtr, videoCount,textDir+"/"+getRandomAudioTextFile(textFiles, int64(videoCount)+123),audioDir+"/"+getRandomAudioTextFile(audioFiles, int64(videoCount)+980), &wg)
				videoCount--
			}
		} else if m == 2 {
			var wg sync.WaitGroup
			for videoCount-procs >= 0 {
				videoOtr:=videoFiles[videoCount-procs:videoCount]
				wg.Add(1)
				go crVideos(videoOtr, videoCount,textDir+"/"+getRandomAudioTextFile(textFiles, int64(videoCount)+123),audioDir+"/"+getRandomAudioTextFile(audioFiles, int64(videoCount)+980), &wg)
				videoCount--
			}
			wg.Wait()
		} else if m == 3 {
			var wg sync.WaitGroup
			gours := os.Args[5]
			g,_:=strconv.Atoi(gours)
			goroutines := make(chan struct{}, g)
			for videoCount-procs >= 0 {
				videoOtr:=videoFiles[videoCount-procs:videoCount]
				wg.Add(1)
				go mixVideos(videoOtr, videoCount,textDir+"/"+getRandomAudioTextFile(textFiles, int64(videoCount)+123),audioDir+"/"+getRandomAudioTextFile(audioFiles, int64(videoCount)+980), &wg, goroutines)
				videoCount--
			}
			wg.Wait()
		}
		fmt.Println("Видео успешно созданы")
		fmt.Printf("Время работы: %v\n", time.Since(start))
		client.Do("HINCRBY", os.Args[1], "use", 1)
	} else {
		fmt.Println("Неверный ключ")
	}

}
///////////////////////
func getFiles(dirPath string, ext string) ([]string, error) {
	var files []string

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && filepath.Ext(path) == ext {
			files = append(files, info.Name())
		}
		return nil
	})

	if err != nil {
		return nil, err
	}
	return files, nil
}

func getRandomAudioTextFile(files []string, sdvig int64) string {
	index := rand.New(rand.NewSource(time.Now().UnixNano()+sdvig)).Intn(len(files))
	return files[index]
}
func mixer(arr []string) [][]string {
	// Инициализируем переменную для хранения всех перестановок
	var permutations [][]string

	// Рекурсивная функция для генерации перестановок
	var permute func(arr []string, start int)

	permute = func(arr []string, start int) {
		// Если достигли конца массива, добавляем текущую перестановку в результат
		if start == len(arr)-1 {
			tmp := make([]string, len(arr))
			copy(tmp, arr)
			permutations = append(permutations, tmp)
		} else {
			// Для каждого элемента, начиная со start, меняем его местами с текущим элементом и рекурсивно вызываем permute
			for i := start; i < len(arr); i++ {
				arr[start], arr[i] = arr[i], arr[start]
				permute(arr, start+1)
				arr[start], arr[i] = arr[i], arr[start] // Восстанавливаем исходное состояние массива
			}
		}
	}

	permute(arr, 0) // Начинаем с индекса 0
	return permutations
}
func mixVideos(videoPaths []string, i int, text string, audioPath string, wg *sync.WaitGroup, quotaChan chan struct{}) {
	quotaChan <- struct{}{}
	defer wg.Done()
	mx:=mixer(videoPaths)
	var wg1 sync.WaitGroup
	for j, perm := range mx {
		wg1.Add(1)
		as, _ :=strconv.Atoi(strconv.Itoa(i)+strconv.Itoa(j))
		go crVideos(perm, as,text,audioPath, &wg1)
	}
	wg1.Wait()
	<-quotaChan
}
func crVideos(videoPaths []string, i int, text string, audioPath string, wg *sync.WaitGroup) error {

	concatFile, err := os.Create("ct"+strconv.Itoa(i)+".txt")
	if err != nil {
		return err
	}
	for _, path := range videoPaths {
		_, err := concatFile.WriteString(fmt.Sprintf("file videos/'%s'\n", path))
		if err != nil {
			return err
		}
	}
	concatFile.Close()
	var outputVideo = "tm"+strconv.Itoa(i)+".mp4"
	cmd := exec.Command(ffmp,
		"-f", "concat",
		"-safe", "0",
		"-i", "ct"+strconv.Itoa(i)+".txt",
		"-c", "copy",
		outputVideo,
	)
	cmd.Run()
	var outputVideo1 = "bm"+strconv.Itoa(i)+".mp4"
	cmd = exec.Command(ffmp,
		"-i", outputVideo,
		"-vf", fmt.Sprintf(`drawtext=textfile='%s':fontfile=font.ttf:fontsize=29:box=0:boxcolor=black@0:boxborderw=5:x=(w-text_w)/2:y=(h-text_h)/2:fontcolor=white`, text),
		"-c:v", "libx264",
		"-preset", "ultrafast",
		"-crf", "18",
		"-b:v", "20M",
		"-c:a", "copy",
		outputVideo1,
	)
	err = cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
	cmd = exec.Command(ffmp,
		"-i", outputVideo1,
		"-i", audioPath,
		"-c:v", "copy",
		"-c:a", "aac",
		"-strict", "-2",
		"-map", "0:v:0",
		"-map", "1:a:0",
		"-shortest",
		"-y",
		fmt.Sprintf("./output/video_%d.mp4", i+1),
	)
	err = cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
	err= os.Remove(outputVideo)
	err= os.Remove(outputVideo1)
	err = os.Remove("ct"+strconv.Itoa(i)+".txt")
	//runtime.GC()
	wg.Done()
	//fmt.Println("Видео с текстом успешно создано")
	return nil
}
func newPool() *redis.Pool {
	return &redis.Pool{
		MaxIdle: 80,
		MaxActive: 12000,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", ":6379", redis.DialPassword(""))

			if err != nil {
				panic(err.Error())
			}
			return c, err
		},
	}
}
