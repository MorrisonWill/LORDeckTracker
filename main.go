package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	g "github.com/AllenDang/giu"
	"github.com/AllenDang/imgui-go"
)

var mulliganed = false

var start_of_game = false

var already_added_deck = false

var in_game = false

var popup_list = []g.Widget{}

var deck_list = [][]string{}

var seen_card_ids = []string{}

var all_cards = [4][]card{}

var cards_in_deck = 0
var one_card_odds = ""
var two_card_odds = ""
var three_card_odds = ""

type card struct {
	AssociatedCards    []string `json:"associatedCards"`
	AssociatedCardRefs []string `json:"associatedCardRefs"`
	Assets             []struct {
		GameAbsolutePath string `json:"gameAbsolutePath"`
		FullAbsolutePath string `json:"fullAbsolutePath"`
	} `json:"assets"`
	Region                string   `json:"region"`
	RegionRef             string   `json:"regionRef"`
	Attack                int      `json:"attack"`
	Cost                  int      `json:"cost"`
	Health                int      `json:"health"`
	Description           string   `json:"description"`
	DescriptionRaw        string   `json:"descriptionRaw"`
	LevelupDescription    string   `json:"levelupDescription"`
	LevelupDescriptionRaw string   `json:"levelupDescriptionRaw"`
	FlavorText            string   `json:"flavorText"`
	ArtistName            string   `json:"artistName"`
	Name                  string   `json:"name"`
	CardCode              string   `json:"cardCode"`
	Keywords              []string `json:"keywords"`
	KeywordRefs           []string `json:"keywordRefs"`
	SpellSpeed            string   `json:"spellSpeed"`
	SpellSpeedRef         string   `json:"spellSpeedRef"`
	Rarity                string   `json:"rarity"`
	RarityRef             string   `json:"rarityRef"`
	Subtype               string   `json:"subtype"`
	Subtypes              []string `json:"subtypes"`
	Supertype             string   `json:"supertype"`
	Type                  string   `json:"type"`
	Collectible           bool     `json:"collectible"`
	Set                   string   `json:"set"`
}

type deck struct {
	DeckCode    string
	CardsInDeck map[string]int
}

type rectangles struct {
	PlayerName   string
	OpponentName string
	GameState    string
	Screen       map[string]int
	Rectangles   []struct {
		CardID      int
		CardCode    string
		TopLeftX    int
		TopLeftY    int
		Width       int
		Height      int
		LocalPlayer bool
	}
}

type result struct {
	GameID         int
	LocalPlayerWon bool
}

const url = "http://127.0.0.1:21337"

func get_game_result() result {
	r, err := http.Get(url + "/game-result")
	if err != nil {
		log.Fatalln(err)
	}
	defer r.Body.Close()

	game_result := result{}
	json.NewDecoder(r.Body).Decode(&game_result)

	return game_result
}

func get_active_deck() deck {
	r, err := http.Get(url + "/static-decklist")
	if err != nil {
		log.Fatalln(err)
	}
	defer r.Body.Close()

	active_deck := deck{}
	json.NewDecoder(r.Body).Decode(&active_deck)

	return active_deck
}

func get_positional_rectangles() rectangles {
	r, err := http.Get(url + "/positional-rectangles")
	if err != nil {
		log.Fatalln(err)
	}
	defer r.Body.Close()

	current_details := rectangles{}
	json.NewDecoder(r.Body).Decode(&current_details)

	return current_details
}

func card_code_to_set(cardCode string) string {
	card_set, _ := strconv.Atoi(cardCode[0:2])

	return fmt.Sprintf(strconv.Itoa(card_set))
}

func card_code_to_name(cardCode string) string {
	card_set, _ := strconv.Atoi(cardCode[0:2])
	// Max set number is four but golang slices start at zero
	set_cards := all_cards[card_set-1]

	for _, v := range set_cards {
		if v.CardCode == cardCode {
			return v.Name
		}
	}

	return ""
}

func update_decklist() {
	if !in_game {
		return
	}

	for {
		active_deck := get_active_deck()

		deck_list = [][]string{}

		for i, j := range active_deck.CardsInDeck {
			deck_list = append(deck_list, []string{i, strconv.Itoa(j)})
		}

		sort.Slice(deck_list, func(p, q int) bool {
			return card_code_to_name(deck_list[p][0]) < card_code_to_name(deck_list[q][0])
		})

		time.Sleep(time.Second)

	}

}

func monitor_played_cards() {
	for in_game {
		rectangles := get_positional_rectangles().Rectangles

		for _, v := range rectangles {

			seen := false
			for _, seen_id_value := range seen_card_ids {
				if strconv.Itoa(v.CardID) == seen_id_value {
					seen = true
				}
			}

			if seen {
				continue
			}

			seen_card_ids = append(seen_card_ids, strconv.Itoa(v.CardID))

			for i, k := range deck_list {
				if v.CardCode == k[0] {
					num_in_deck, _ := strconv.Atoi(deck_list[i][1])
					num_in_deck--
					deck_list[i][1] = strconv.Itoa(num_in_deck)
				}
			}
		}

		time.Sleep(time.Second)
	}
}

func monitor_card_odds() {
	for in_game {
		time.Sleep(time.Second)

		cards_in_deck = num_cards_in_deck()

		float_cards_in_deck := float64(cards_in_deck)
		// one_card_odds = math.Round(1 / float_cards_in_deck * 100)

		one_card_odds = float_odds_to_string(1 / float_cards_in_deck * 100)
		two_card_odds = float_odds_to_string(2 / float_cards_in_deck * 100)
		three_card_odds = float_odds_to_string(3 / float_cards_in_deck * 100)

	}

}

func num_cards_in_deck() int {

	cards_in_deck := 0
	for _, v := range deck_list {
		int_num, _ := strconv.Atoi(v[1])
		cards_in_deck += int_num
	}
	return cards_in_deck
}

func float_odds_to_string(input_num float64) string {
	s := strconv.FormatFloat(input_num, 'f', 6, 64)
	split := strings.Split(s, ".")

	return s[0 : len(split[0])+3]
}

func build_rows() []*g.TableRowWidget {
	rows := make([]*g.TableRowWidget, len(deck_list))

	// rows[0] = g.TableRow(
	// 	g.Label("Card name"),
	// 	g.Label("Number left in deck"),
	// )

	for i := range rows {
		index_copy := i
		current_card := deck_list[index_copy][0]
		rows[i] = g.TableRow(g.Column(
			g.Popup(current_card).Layout(g.ImageWithFile(card_code_to_file_path(current_card)).Size(226.667, 341.333)),
			g.Selectable(card_code_to_name(deck_list[i][0])).OnClick(func() { g.OpenPopup(current_card) })),
			g.Label(deck_list[i][1]),
		)
	}

	return rows
}

func card_code_to_file_path(input string) string {
	set_number := card_code_to_set(input)
	file_path := fmt.Sprintf("set_data/set%s-en_us/en_us/img/cards/%s.png", set_number, input)

	return file_path
}

func loop() {
	g.SingleWindow("Deck Tracker Window").Layout(
		g.Label("Legends of Runeterra Deck Tracker"),
		g.Table("Deck List Table").Flags(imgui.TableFlags_Borders).Rows(build_rows()...),
		g.Label(fmt.Sprintf("One: %s%%, Two: %s%%, Three: %s%%", one_card_odds, two_card_odds, three_card_odds)),
		g.Button("Done With Mulligan").OnClick(func() { mulliganed = true }),
		g.Label("Created By Will Morrison"),
	)

}

func manage_mulligan() {
	for !mulliganed {
		time.Sleep(time.Second)
	}
	go monitor_played_cards()
	go monitor_card_odds()
}

func watch_game_state() {
	for {
		game_state := get_positional_rectangles().GameState
		if game_state != "InProgress" {
			already_added_deck = false
			in_game = false
			time.Sleep(time.Second)
			continue
		}

		in_game = true

		if !already_added_deck {
			one_card_odds = "0"
			two_card_odds = "0"
			three_card_odds = "0"
			mulliganed = false
			already_added_deck = true
			active_deck := get_active_deck()

			deck_list = [][]string{}

			for i, j := range active_deck.CardsInDeck {
				deck_list = append(deck_list, []string{i, strconv.Itoa(j)})
			}

			sort.Slice(deck_list, func(p, q int) bool {
				return card_code_to_name(deck_list[p][0]) < card_code_to_name(deck_list[q][0])
			})

			for _, v := range deck_list {
				card_name := v[0]
				set_number := card_code_to_set(card_name)
				file_path := fmt.Sprintf("set_data/set%s-en_us/en_us/img/%s.png", set_number, card_name)
				popup_list = []g.Widget{}
				popup_list = append(popup_list, g.Popup(card_name).Layout(g.ImageWithFile(file_path).Size(400, 600)))
			}
		}

		go manage_mulligan()

		time.Sleep(time.Second)
	}
}

func main() {
	for i := range all_cards {
		// sets start at 1, not 0
		current_set := strconv.Itoa(i + 1)

		json_file_path := "set_data/set" + current_set + "-en_us/en_us/data/set" + current_set + "-en_us.json"

		content, err := ioutil.ReadFile(json_file_path)
		if err != nil {
			log.Fatalln("Error when opening file: ", err)
		}
		err = json.Unmarshal(content, &all_cards[i])
		if err != nil {
			log.Fatal("Error during Unmarshal(): ", err)
		}
	}

	go watch_game_state()

	wnd := g.NewMasterWindow("LOR Deck Tracker", 415, 375, g.MasterWindowFlagsFloating, nil)
	wnd.Run(loop)
}

// TODO settings?
// TODO add goroutine for monitoring if in game or menus and manage the deck tracking goroutine from that. Turn off with channel.
// TODO clean up all code
// TODO poll rectangle positions fast enough to see card draws and mulligans (look at githubs for ideas)
