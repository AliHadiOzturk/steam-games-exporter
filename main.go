package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/google/uuid"
)

type AppList struct {
	Response struct {
		Apps      []*App `json:"apps"`
		HaveMore  bool   `json:"have_more_results"`
		LastAppId int    `json:"last_appid"`
	} `json:"response"`
}

type App struct {
	Id         int    `json:"appid"`
	Name       string `json:"name"`
	ProviderId string `json:"-"`
}

type SupabaseGame struct {
	Id           string `json:"id"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	Icon         string `json:"icon"`
	IconLocation string `json:"icon_location"`
	Code         string `json:"code"`
	PlatformId   string `json:"platform_id"`
}

var accessToken = "ACCESS TOKEN FROM STEAM"

var lastAppId = 0

func main() {

	file, err := os.ReadFile("last_appid.txt")

	if err != nil {
		fmt.Println("cannot read file", err)
	}

	lastAppId, _ = strconv.Atoi(string(file))

	csvFile, err := os.OpenFile("games.csv", os.O_APPEND|os.O_CREATE|os.O_WRONLY, os.ModePerm)
	if err != nil {
		log.Fatalf("failed creating file: %s", err)
	}

	defer csvFile.Close()

	csvwriter := csv.NewWriter(csvFile)

	csvwriter.Write([]string{"id", "name", "description", "icon", "code", "icon_location", "platform_id"})

	csvwriter.Flush()

	var appList *AppList

	for {

		// if appList.Response.HaveMore == false {
		// 	break
		// }

		// if appList.Response.LastAppId == 0 {
		// 	break
		// }

		res, err := http.Get(fmt.Sprintf("https://api.steampowered.com/IStoreService/GetAppList/v1/?access_token=%s&include_games=true&max_results=50000&last_appid=%d", accessToken, lastAppId))

		if err != nil {
			fmt.Println("cannot get games", err)
		}

		body, _ := io.ReadAll(res.Body)

		if err = json.Unmarshal(body, &appList); err != nil {
			fmt.Println("cannot unmarshal response", err)
			panic(err)
		}

		if len(appList.Response.Apps) == 0 {
			break
		}

		// if err = json.Unmarshal(apps, &list); err != nil {
		// 	fmt.Println("cannot unmarshal games", err)
		// 	panic(err)
		// }

		var games []SupabaseGame

		for index, game := range appList.Response.Apps {

			if game.Id == 0 || game.Name == "" {
				continue
			}

			if index%100 == 0 {
				fmt.Printf("Progress %d \n", index)
			}

			res, err := http.Get(fmt.Sprintf("https://store.steampowered.com/api/appdetails?appids=%d", game.Id))

			if err != nil {
				fmt.Println("cannot get game", err)

			}

			body, _ := io.ReadAll(res.Body)

			var r map[string]interface{}

			if err = json.Unmarshal(body, &r); err != nil {
				continue
			}

			gameId := fmt.Sprint(game.Id)

			if r[gameId] == nil {
				fmt.Println("Game not found", game.Id)
				continue
			}

			success := r[gameId].(map[string]interface{})["success"].(bool)

			if success == false {
				fmt.Println("Game not found", game.Id)
				continue
			}

			gameData := r[gameId].(map[string]interface{})["data"].(map[string]interface{})

			data := &SupabaseGame{
				Id:           uuid.NewString(),
				Name:         game.Name,
				Description:  gameData["short_description"].(string),
				Icon:         gameData["header_image"].(string),
				IconLocation: "remote",
				Code:         gameId,
				PlatformId:   "48dcdb32-aafe-4d22-a9a8-9e55f926f32b",
			}

			appInfo, err := http.Get(fmt.Sprintf("https://api.steampowered.com/ICommunityService/GetApps/v1/?appids[0]=%s", strconv.Itoa(game.Id)))

			if err != nil {
				fmt.Println("cannot get game", err)
				continue
			}

			appInfoBody, _ := io.ReadAll(appInfo.Body)

			var appInfoResponse map[string]interface{}

			if err = json.Unmarshal(appInfoBody, &appInfoResponse); err != nil {
				fmt.Println("cannot unmarshal app info", err)
				continue
			}

			if appInfoResponse["response"] == nil {
				fmt.Println("App info not found", game.Id)
				continue
			}

			apps := appInfoResponse["response"].(map[string]interface{})["apps"].([]interface{})

			icon := apps[0].(map[string]interface{})["icon"]

			data.Icon = fmt.Sprintf("https://cdn.cloudflare.steamstatic.com/steamcommunity/public/images/apps/%s/%s.jpg", gameId, icon)

			lastAppId = game.Id

			file, err := os.Create("last_appid.txt")

			if err != nil {
				fmt.Println("cannot create file", err)
			}

			file.WriteString(strconv.Itoa(lastAppId))

			file.Close()

			csvwriter.Write([]string{data.Id, data.Name, data.Description, data.Icon, data.Code, data.IconLocation, data.PlatformId})
			csvwriter.Flush()

			games = append(games, *data)
		}

		fmt.Printf("Total count: %d\n", len(games))
	}

}
