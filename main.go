package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/Fefefo/moeScraper/scraper"

	"github.com/Knetic/govaluate"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"gopkg.in/ini.v1"
)

var tgapi, filmapi, catapi, transapi string
var bot *tgbotapi.BotAPI

type transBody struct {
	Translations []translations `json:"translations,omitempty"`
}

type translations struct {
	Text string `json:"text,omitempty"`
	To   string `json:"to,omitempty"`
}

type dict struct {
	Word     string                  `json:"word,omitempty"`
	Phonetic string                  `json:"phonetic,omitempty"`
	Origin   string                  `json:"origin,omitempty"`
	Meaning  map[string][]definition `json:"meaning,omitempty"`
}

type definition struct {
	Definition string   `json:"definition,omitempty"`
	Example    string   `json:"example,omitempty"`
	Synonyms   []string `json:"synonym,omitempty"`
}

type gattoLink struct {
	Link string `json:"url"`
}

type caneLink struct {
	Link string `json:"message"`
}

type film struct {
	Results []struct {
		//VoteAverage float64 `json:"vote_average"`
		ReleaseDate string `json:"release_date"`
		Title       string `json:"title"`
		Overview    string `json:"overview"`
	}
}

type urbanResponse struct {
	List []struct {
		Def         string        `json:"definition"`
		Permalink   string        `json:"permalink"`
		ThumbsUp    int           `json:"thumbs_up"`
		SoundUrls   []interface{} `json:"sound_urls"`
		Author      string        `json:"author"`
		Word        string        `json:"word"`
		Defid       int           `json:"defid"`
		CurrentVote string        `json:"current_vote"`
		WrittenOn   time.Time     `json:"written_on"`
		Example     string        `json:"example"`
		ThumbsDown  int           `json:"thumbs_down"`
	} `json:"list"`
}

type redditResponse struct {
	Data struct {
		Children []struct {
			Data struct {
				File string `json:"url_overridden_by_dest"`
				//Thumbnail string `json:"thumbnail"`
				Title     string `json:"title"`
				Subreddit string `json:"subreddit_name_prefixed"`
			} `json:"data"`
		} `json:"children"`
	} `json:"data"`
}

func getUrb(query string) urbanResponse {
	link := "http://api.urbandictionary.com/v0/define?term=" + url.PathEscape(query)

	req, _ := http.Get(link)

	body, _ := ioutil.ReadAll(req.Body)

	var res urbanResponse
	json.Unmarshal(body, &res)
	return res
}

func getReddit(query string) []redditResponse {
	if strings.HasPrefix(query, "www") {
		query = "https://" + query
	}

	if !strings.HasPrefix(query, "https://www.reddit.com/r/") && !strings.HasPrefix(query, "http://www.reddit.com/r/") {
		return nil
	}

	client := &http.Client{}
	var res []redditResponse

	splitted := strings.Split(query, "/")
	if len(splitted) < 8 {
		return nil
	}

	query = strings.Join(splitted[:8], "/")
	query += ".json"

	req, err := http.NewRequest("GET", query, nil)
	req.Header.Set("User-Agent", "Golang_FefegoBot")
	if err != nil {
		return nil
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil
	}

	body, err := ioutil.ReadAll(resp.Body)
	if fmt.Sprintf("%s", body) == "[]" || err != nil {
		return nil
	}

	err = json.Unmarshal(body, &res)
	if err != nil || len(res) == 0 || len(res[0].Data.Children) == 0 {
		return nil
	}

	return res
}

func translate(text, to string) string {
	link := "https://microsoft-translator-text.p.rapidapi.com/translate?profanityAction=NoAction&textType=plain&api-version=3.0&to=" + url.PathEscape(to)

	txt := strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(text, "\r", ""), "\n", ""), "\\", "/"), "\"", "'")
	reqText := strings.NewReader(`[{"Text": "` + txt + `" }]`)

	req, _ := http.NewRequest("POST", link, reqText)

	req.Header.Add("x-rapidapi-host", "microsoft-translator-text.p.rapidapi.com")
	req.Header.Add("x-rapidapi-key", transapi)
	req.Header.Add("content-type", "application/json")

	client, _ := http.DefaultClient.Do(req)
	body, _ := ioutil.ReadAll(client.Body)
	var res []transBody
	json.Unmarshal(body, &res)
	fmt.Println(client)
	fmt.Println(string(body))
	var out string
	if len(res) > 0 {
		out = res[0].Translations[0].Text
	} else {
		out = "Errore, utilizza la giusta sintassi del comando `.transto [lingua]` utilizzando il codice ISO della lingua in cui vuoi tradurre il messaggio (es: `it, en, es`)"
	}
	return out + "."
}

func getGatto(api string) string {
	resp, _ := http.Get("https://api.thecatapi.com/v1/images/search?api_key=" + api)
	body, _ := ioutil.ReadAll(resp.Body)
	var gattino []gattoLink
	json.Unmarshal([]byte(body), &gattino)
	return gattino[0].Link
}

func getCane() (string, string) {
	respC, _ := http.Get("https://dog.ceo/api/breeds/image/random")
	bodyC, _ := ioutil.ReadAll(respC.Body)
	var cagnolino caneLink
	json.Unmarshal([]byte(bodyC), &cagnolino)
	return cagnolino.Link, "Cane di razza: " + strings.Split(strings.Replace(cagnolino.Link, "https://images.dog.ceo/breeds/", "", -1), "/")[0]
}

func getTime(start time.Time) string {
	temp := time.Now()
	diff := temp.Sub(start)
	s := int(diff.Seconds())
	d := s / (3600 * 24)
	s = s % (3600 * 24)
	h := s / 3600
	s = s % 3600
	m := s / 60
	s = s % 60
	out := ""
	if m > 0 {
		if h > 0 {
			if d > 0 {
				out += strconv.Itoa(d) + "d "
			}
			out += strconv.Itoa(h) + "h "
		}
		out += strconv.Itoa(m) + "m "
	}
	out += strconv.Itoa(s) + "s"
	return out
}

func solver(mate string) string {
	expression, er := govaluate.NewEvaluableExpression(mate)
	testo := fmt.Sprintf(`Il risultato di <code>%s</code> è
	<code>`, mate)
	if er == nil {
		parameters := make(map[string]interface{}, 8)
		result, err := expression.Evaluate(parameters)
		if err == nil {
			if fmt.Sprintf("%T", result) == "float64" {
				if math.Mod(result.(float64), 1) == 0 {
					return fmt.Sprintf("%s%.0f</code>", testo, result)
				} else {
					return fmt.Sprintf("%s%0.3f</code>", testo, result)
				}
			}
			return fmt.Sprintf("%s%v</code>", testo, result)
		} else {
			return fmt.Sprintf(`Espressione : <code>%s</code>
			
			Non so come risolvere l'espressione`, mate)
		}
	} else {
		return fmt.Sprintf(`Espressione : <code>%s</code>
		
		Non riesco a capire la tua sintassi`, mate)
	}
}

func getAnimeList() scraper.List {
	animeList := scraper.GetAnimeList()
	files := 0
	for i := 0; i < len(animeList); i++ {
		files += len(animeList[i].Songs)
	}
	msg := tgbotapi.NewMessage(146519367, fmt.Sprintf("Loaded %d anime (%d files)", len(animeList), files))
	bot.Send(msg)
	return animeList
}

func main() {
	cfg, err := ini.Load("my.ini")
	if err != nil {
		fmt.Printf("Fail to read file: %v", err)
	}

	tgapi = cfg.Section("").Key("tgbot_api").String()
	filmapi = cfg.Section("").Key("film_api").String()
	catapi = cfg.Section("").Key("cat_api").String()
	transapi = cfg.Section("").Key("translate_api").String()
	myid, _ := cfg.Section("").Key("my_id").Int()
	channelid, _ := cfg.Section("").Key("channel_id").Int64()

	bot, err = tgbotapi.NewBotAPI(tgapi)
	if err != nil {
		log.Panic(err)
	}

	//bot.Debug = true
	startTime := time.Now()

	const helpText string = "*Il bot ha alcuni comandi utilizzabili con la seguente sintassi*\n" +
		"`@fefegobot 'key' 'query'`\n\n" +
		"*Key al momento disponibili:*\n" +
		"`parola:` verrà cercata una parola italiana in un dizionario\n" +
		"	*es.* `@fefegobot parola ciao`\n\n" +
		"`word:` verrà cercata una parola inglese in un dizionario\n" +
		"	*es.* `@fefegobot word hello`\n\n" +
		"`film:` verranno cercati film con le parole inserite\n" +
		"	*es.* `@fefegobot film The Shape of Voice`\n\n" +
		"`math:` verrà eseguita l'espressione immessa\n" +
		"	*es.* `@fefegobot math 5+6*7-2/(5-77)`\n\n" +
		"`neko:` verrà cercata una foto di un gatto con le risoluzioni inserite\n" +
		"	*es.* `@fefegobot neko 1920 1080`\n\n" +
		"`reddimg:` verrà mandata la foto di un post reddit\n" +
		"	*es.* `@fefegobot reddimg https://www.reddit.com/r/CatsAreAssholes/comments/kj6a90/we_dont_need_a_christmass_tree_topper_this_year/`\n\n" +
		"`theme:` verranno cercate tutte le opening e ending di anime che contengono i 5 caratteri inseriti nel nome\n" +
		"	*es.* `@fefegobot theme evang`\n\n\n" +
		"*È inoltre possibile mandare le proprie foto del profilo scrivendo :*\n" +
		"`@fefegobot mypics`\n\n" +
		"*e il proprio id :*\n" +
		"`@fefegobot myid`\n\n" +
		"_Bot online da_ `"

	log.Printf("Authorized on account %s", bot.Self.UserName)

	animeList := getAnimeList()

	log.Println("Loaded", len(animeList), "anime")
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 5

	updates, err := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.InlineQuery != nil {

			rand.Seed(time.Now().UTC().UnixNano())
			var array []interface{}

			inlineText := update.InlineQuery.Query
			splittedText := strings.Split(inlineText, " ")

			if splittedText[0] == "parola" && len(splittedText) == 2 {
				url := "https://api.dictionaryapi.dev/api/v1/entries/it/" + splittedText[1]
				resp, _ := http.Get(url)
				body, _ := ioutil.ReadAll(resp.Body)

				var query []dict

				json.Unmarshal([]byte(body), &query)

				if len(query) != 0 {
					i := 0
					for j := range query {
						for tipo, means := range query[j].Meaning {
							//log.Println("Tipo:", tipo)
							for _, def := range means {
								text := "*Parola ricercata:* `" + query[j].Word + "`\n" +
									"*Divisione in sillabe:* `" + query[j].Phonetic + "`\n" +
									"*Significato:*\n" +
									"`" + def.Definition + "`"
								articolo := tgbotapi.NewInlineQueryResultArticleMarkdown("significato"+fmt.Sprintf("%d", i), query[j].Word, text)
								articolo.Description = tipo
								array = append(array, articolo)
								i++
							}
						}
					}
				}
				if len(array) == 0 {
					articolo := tgbotapi.NewInlineQueryResultPhotoWithThumb("404", "https://http.cat/404", "https://http.cat/404")
					articolo.Caption = "Nessun parola italiana trovata cercando " + splittedText[1]
					articolo.Description = "NOPE"
					array = append(array, articolo)
				}
			} else if splittedText[0] == "word" && len(splittedText) == 2 {
				url := "https://api.dictionaryapi.dev/api/v1/entries/en/" + splittedText[1]
				resp, _ := http.Get(url)
				body, _ := ioutil.ReadAll(resp.Body)

				var query []dict

				json.Unmarshal([]byte(body), &query)

				if len(query) != 0 {
					i := 0
					for tipo, means := range query[0].Meaning {
						for _, def := range means {
							sinonimi := ""
							for i := 0; i < len(def.Synonyms); i++ {
								if sinonimi != "" {
									sinonimi += " " + def.Synonyms[i]
								} else {
									sinonimi = def.Synonyms[0]
								}
							}
							text := "*Word:* `" + splittedText[1] + "`\n" +
								"*Phonetic:* `" + query[0].Phonetic + "`\n" +
								"*Synonyms:*\n" +
								"`" + sinonimi + "`\n" +
								"*Definition:*\n" +
								"`" + def.Definition + "`"
							articolo := tgbotapi.NewInlineQueryResultArticleMarkdown("mean"+fmt.Sprintf("%d", i), splittedText[1], text)
							articolo.Description = tipo
							array = append(array, articolo)
							i++
						}
					}
				}
				if len(array) == 0 {
					articolo := tgbotapi.NewInlineQueryResultPhotoWithThumb("404", "https://http.cat/404", "https://http.cat/404")
					articolo.Caption = "Nessun parola inglese trovata cercando " + splittedText[1]
					articolo.Description = "NOPE"
					array = append(array, articolo)
				}
			} else if splittedText[0] == "film" && len(splittedText) >= 2 {
				ricerca := url.QueryEscape(strings.Join(splittedText[1:], " "))
				url := "https://api.themoviedb.org/3/search/movie?api_key=" + filmapi + "&language=it&query=" + ricerca + "&page=1&include_adult=true"

				resp, _ := http.Get(url)
				body, _ := ioutil.ReadAll(resp.Body)

				var query film
				json.Unmarshal([]byte(body), &query)
				i := 0

				for _, n := range query.Results {
					articolo := tgbotapi.NewInlineQueryResultArticleMarkdown("film"+fmt.Sprintf("%d", i), n.Title, "***"+n.Title+"***" /*+"  "+fmt.Sprintf("%.1f", n.VoteAverage)+" ⭐\n"*/ +"\n"+"`"+n.ReleaseDate+"`\n"+n.Overview)
					articolo.Description = n.ReleaseDate
					array = append(array, articolo)
					i++
				}
				if len(array) == 0 {
					articolo := tgbotapi.NewInlineQueryResultPhotoWithThumb("404", "https://http.cat/404", "https://http.cat/404")
					articolo.Caption = "Nessun film trovato cercando " + ricerca
					articolo.Description = "NOPE"
					array = append(array, articolo)
				}

			} else if splittedText[0] == "math" && len(splittedText) >= 2 {
				formula := strings.Join(splittedText[1:], " ")
				if formula != "" {
					sol := solver(formula)
					articolo := tgbotapi.NewInlineQueryResultArticleHTML("soluzione", "Risolvi", sol)
					articolo.Description = formula
					array = append(array, articolo)
				}
			} else if splittedText[0] == "neko" && len(splittedText) == 3 {
				url := "http://placekitten.com/" + splittedText[1] + "/" + splittedText[2]
				resp, _ := http.Get(url)
				if resp.StatusCode == 200 {
					gattosize := tgbotapi.NewInlineQueryResultPhotoWithThumb("nekoSize", url, "http://image.thepaper.cn/www/image/28/100/499.jpg")
					gattosize.Description = "Neko 猫"
					gattosize.Caption = "Neko 猫\n" +
						"Risoluzione : " + splittedText[1] + " x " + splittedText[2] + " px"
					array = append(array, gattosize)
				} else if len(array) == 0 {
					articolo := tgbotapi.NewInlineQueryResultPhotoWithThumb("404", "https://http.cat/404", "https://http.cat/404")
					articolo.Caption = "Non sono riuscito a trovare foto di gatti larghe " + splittedText[1] + " px e alte " + splittedText[2] + " px"
					articolo.Description = "NOPE"
					array = append(array, articolo)
				}
			} else if splittedText[0] == "theme" && len(splittedText) >= 2 {
				query := strings.Join(splittedText[1:], " ")
				if len(query) >= 3 {
					lista := animeList.SelectByBothNames(query)
					count := 0
					for i := 0; i < len(lista) && count < 50; i++ {
						for j := 0; j < len(lista[i].Songs) && count < 50; j++ {
							esempio := tgbotapi.NewInlineQueryResultArticleHTML("animesong"+strconv.Itoa(i)+"id"+strconv.Itoa(j), lista[i].NameJap, lista[i].NameJap+"\n"+lista[i].NameEng+"\n"+"<a href='"+lista[i].Songs[j].Link+"'>"+lista[i].Songs[j].Version+" - "+lista[i].Songs[j].Title+"</a>")
							esempio.Description = strings.Split(lista[i].NameEng, ",")[0] + " - " + lista[i].Songs[j].Version + " - " + lista[i].Songs[j].Title
							array = append(array, esempio)
							count++
						}
					}

					if len(array) == 0 {
						articolo := tgbotapi.NewInlineQueryResultPhotoWithThumb("404", "https://http.cat/404", "https://http.cat/404")
						articolo.Caption = "Nessuna opening o ending trovata cercando " + query
						articolo.Description = "NOPE"
						array = append(array, articolo)
					}
				}
			} else if splittedText[0] == "mypics" {
				conf := tgbotapi.UserProfilePhotosConfig{
					UserID: update.InlineQuery.From.ID,
				}
				foto, _ := bot.GetUserProfilePhotos(conf)
				page := 0
				if len(splittedText) == 2 {
					if num, err := strconv.Atoi(splittedText[1]); err == nil && int(len(foto.Photos)/50) > (num-1) {
						page = num - 1
					}
				}
				for i := page * 50; i < len(foto.Photos); i++ {
					if i < (page+1)*50 {
						a := tgbotapi.NewInlineQueryResultPhotoWithThumb("pic"+strconv.Itoa(i), foto.Photos[i][2].FileID, foto.Photos[i][2].FileID)
						a.Caption = "La tua pic numero " + strconv.Itoa(i+1)
						array = append(array, a)
					}
				}
			} else if splittedText[0] == "urban" && update.InlineQuery.From.ID == myid {
				if len(splittedText) > 1 {
					query := strings.Join(splittedText[1:], " ")
					if strings.HasSuffix(query, ".") {
						urb := getUrb(strings.ReplaceAll(query, ".", ""))
						for i := 0; i < len(urb.List); i++ {
							//a := tgbotapi.NewInlineQueryResultArticleHTML("urban"+strconv.Itoa(i), urb.List[i].Word, "Title: <a href='"+urb.List[i].Permalink+"'>"+urb.List[i].Word+"</a>\n\n"+strings.ReplaceAll(strings.ReplaceAll(urb.List[i].Def, "[", ""), "]", "")+"\n\nAuthor: "+urb.List[i].Author)
							a := tgbotapi.NewInlineQueryResultArticle("urban"+strconv.Itoa(i), urb.List[i].Word, "Title: "+urb.List[i].Word+"\n\n"+strings.ReplaceAll(strings.ReplaceAll(urb.List[i].Def, "[", ""), "]", "")+"\n\nAuthor: "+urb.List[i].Author)
							a.Description = urb.List[i].Author
							array = append(array, a)
						}
					}
				}
			} else if splittedText[0] == "reddimg" {
				if len(splittedText) == 2 {
					query := splittedText[1]
					img := getReddit(query)
					if img != nil {
						msg := tgbotapi.NewDocumentUpload(channelid, nil)
						msg.FileID = img[0].Data.Children[0].Data.File
						msg.UseExisting = true
						msg.ParseMode = tgbotapi.ModeHTML
						msg.Caption = "<a href='" + query + "'>Sauce</a>\nby@" + update.InlineQuery.From.UserName
						m, err := bot.Send(msg)

						if err == nil {
							a := tgbotapi.NewInlineQueryResultCachedDocument("img", m.Document.FileID, img[0].Data.Children[0].Data.Title)
							a.Caption = img[0].Data.Children[0].Data.Title + "\n<a href='" + query + "'>Sauce</a> from <a href='reddit.com/" + img[0].Data.Children[0].Data.Subreddit + "'>" + img[0].Data.Children[0].Data.Subreddit + "</a>"
							a.ParseMode = tgbotapi.ModeHTML
							array = append(array, a)
						} else {
							a := tgbotapi.NewInlineQueryResultArticleHTML("sad", img[0].Data.Children[0].Data.Title, "Link al <a href='"+query+"'>POST</a>\nLink all'<a href='"+img[0].Data.Children[0].Data.File+"'>IMMAGINE</a>")
							a.Description = "File probabilmente troppo grande, puoi comunque mandare il link cliccando qui"
							array = append(array, a)
						}
					} else {
						articolo := tgbotapi.NewInlineQueryResultPhotoWithThumb("404", "https://http.cat/404", "https://http.cat/404")
						articolo.Caption = "Hai mandato un link sbagliato"
						articolo.Description = "NOPE"
						array = append(array, articolo)
					}
				}
			} else if splittedText[0] == "myid" {
				a := tgbotapi.NewInlineQueryResultArticleMarkdown("id", update.InlineQuery.From.FirstName+" ID", update.InlineQuery.From.FirstName+" id = `"+strconv.Itoa(update.InlineQuery.From.ID)+"`")
				a.Description = strconv.Itoa(update.InlineQuery.From.ID)

				array = append(array, a)
			} else if splittedText[0] == "relThemes" && update.InlineQuery.From.ID == myid {
				animeList = getAnimeList()
			} else {
				stats := tgbotapi.NewInlineQueryResultArticleMarkdown("help", "HELP", helpText+getTime(startTime)+"`")
				stats.Description = "Comandi aggiuntivi e tempo online"
				stats.ThumbURL = "https://pngimage.net/wp-content/uploads/2018/05/chiave-inglese-png-1.png"

				ping := tgbotapi.NewInlineQueryResultArticle("ping", "PING", "PONG")
				ping.Description = "Latenza? O FORSE NO"
				ping.ThumbURL = "https://www.dictionary.com/e/wp-content/uploads/2018/03/eyes-1.jpg"

				gatto := tgbotapi.NewInlineQueryResultPhotoWithThumb("neko", getGatto(catapi), "http://image.thepaper.cn/www/image/28/100/499.jpg")
				gatto.Description = "Neko 猫"
				gatto.Caption = "Neko 猫"

				caneLink, caneCaption := getCane()
				cane := tgbotapi.NewInlineQueryResultPhotoWithThumb("inu", caneLink, "https://dog.ceo/img/dog-api-logo.svg")
				cane.Description = "Inu 犬"
				cane.Caption = caneCaption

				array = append(array, stats, ping, gatto, cane)
			}

			risposta := tgbotapi.InlineConfig{
				InlineQueryID:     update.InlineQuery.ID,
				IsPersonal:        false,
				CacheTime:         0,
				Results:           array,
				SwitchPMText:      "Bot di @Fedefio",
				SwitchPMParameter: "255",
			}

			if _, err = bot.AnswerInlineQuery(risposta); err != nil {
				log.Println(err)
			}
			bot.AnswerInlineQuery(risposta)
		} else if update.Message != nil {
			if strings.HasPrefix(update.Message.Text, ".transto") && update.Message.ReplyToMessage != nil {
				to := strings.TrimSpace(strings.Replace(update.Message.Text, ".transto", "", 1))
				out := translate(update.Message.ReplyToMessage.Text, to)
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, out)
				msg.ReplyToMessageID = update.Message.ReplyToMessage.MessageID
				bot.Send(msg)
			}
		}
	}
}
