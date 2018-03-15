package biasgame

import (
	"fmt"
	"sort"
	"strings"

	"github.com/Snakeyesz/snek-bot/cache"
	"github.com/Snakeyesz/snek-bot/models"
	"github.com/Snakeyesz/snek-bot/utils"
	"github.com/bwmarrin/discordgo"
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
)

// displayBiasGameStats will display stats for the bias game based on the stats message
func displayBiasGameStats(msg *discordgo.Message, statsMessage string) {
	results, iconURL, targetName := getStatsResults(msg, statsMessage)

	// check if any stats were returned
	totalGames, err := results.Count()
	if err != nil || totalGames == 0 {
		utils.SendMessage(msg.ChannelID, "biasgame.stats.no-stats")
		return
	}

	statsTitle := ""
	countsHeader := ""

	// loop through the results and compile a map of [biasgroup biasname]number of occurences
	items := results.Iter()
	biasCounts := make(map[string]int)
	game := models.BiasGameEntry{}
	for items.Next(&game) {
		groupAndName := ""

		if strings.Contains(statsMessage, "rounds won") {

			// round winners
			for _, rWinner := range game.RoundWinners {

				if strings.Contains(statsMessage, "group") {
					statsTitle = "Rounds Won in Bias Game by Group"
					groupAndName = fmt.Sprintf("%s", rWinner.GroupName)
				} else {
					statsTitle = "Rounds Won in Bias Game"
					groupAndName = fmt.Sprintf("**%s** %s", rWinner.GroupName, rWinner.Name)
				}
				biasCounts[groupAndName] += 1
			}

			countsHeader = "Rounds Won"

		} else if strings.Contains(statsMessage, "rounds lost") {

			// round losers
			for _, rLoser := range game.RoundLosers {

				if strings.Contains(statsMessage, "group") {
					statsTitle = "Rounds Lost in Bias Game by Group"
					groupAndName = fmt.Sprintf("%s", rLoser.GroupName)
				} else {
					statsTitle = "Rounds Lost in Bias Game"
					groupAndName = fmt.Sprintf("**%s** %s", rLoser.GroupName, rLoser.Name)
				}
				biasCounts[groupAndName] += 1
			}

			statsTitle = "Rounds Lost in Bias Game"
			countsHeader = "Rounds Lost"
		} else {

			// game winners
			if strings.Contains(statsMessage, "group") {
				statsTitle = "Bias Game Winners by Group"
				groupAndName = fmt.Sprintf("%s", game.GameWinner.GroupName)
			} else {
				statsTitle = "Bias Game Winners"
				groupAndName = fmt.Sprintf("**%s** %s", game.GameWinner.GroupName, game.GameWinner.Name)
			}

			biasCounts[groupAndName] += 1
			countsHeader = "Games Won"

		}
	}

	// add total games to the stats header message
	statsTitle = fmt.Sprintf("%s (%d games)", statsTitle, totalGames)

	sendStatsMessage(msg, statsTitle, countsHeader, biasCounts, iconURL, targetName)
}

// listIdolsInGame will list all idols that can show up in the biasgame
func listIdolsInGame(msg *discordgo.Message) {

	// create map of idols and there group
	groupIdolMap := make(map[string][]string)
	for _, bias := range allBiasChoices {
		if len(bias.biasImages) > 1 {

			groupIdolMap[bias.groupName] = append(groupIdolMap[bias.groupName], fmt.Sprintf("%s (%d)", bias.biasName, len(bias.biasImages)))
		} else {

			groupIdolMap[bias.groupName] = append(groupIdolMap[bias.groupName], fmt.Sprintf("%s", bias.biasName))
		}
	}

	embed := &discordgo.MessageEmbed{
		Color: 0x0FADED, // blueish
		Author: &discordgo.MessageEmbedAuthor{
			Name: fmt.Sprintf("All Idols Available In Bias Game (%d total)", len(allBiasChoices)),
		},
		Title: "*Numbers indicate multi pictures are available for the idol*",
	}

	// make fields for each group and the idols in the group.
	for group, idols := range groupIdolMap {

		// sort idols by name
		sort.Slice(idols, func(i, j int) bool {
			return idols[i] < idols[j]
		})

		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   group,
			Value:  strings.Join(idols, ", "),
			Inline: false,
		})
	}

	// sort fields by group name
	sort.Slice(embed.Fields, func(i, j int) bool {
		return strings.ToLower(embed.Fields[i].Name) < strings.ToLower(embed.Fields[j].Name)
	})

	utils.SendPagedMessage(msg, embed, 10)
}

// listIdolsInGame will list all idols that can show up in the biasgame
func showSingleGameRankings(msg *discordgo.Message) {

	fmt.Println("Getting Rankings...")
	type rankingStruct struct {
		userId           string
		amountOfGames    int
		idolWithMostWins string
		userName         string
	}

	results := utils.MongoDBSearch(models.BiasGameTable, bson.M{"gametype": "single"})

	// check if any stats were returned
	totalGames, err := results.Count()
	if err != nil || totalGames == 0 {
		utils.SendMessage(msg.ChannelID, "biasgame.stats.no-stats")
		return
	}

	fmt.Println("Result Count: ", totalGames)

	// loop through the results and compile a map of userids => gameWinner group+name
	items := results.Iter()
	rankingsInfo := make(map[string][]string)
	game := models.BiasGameEntry{}
	for items.Next(&game) {
		rankingsInfo[game.UserID] = append(rankingsInfo[game.UserID], fmt.Sprintf("%s %s", game.GameWinner.GroupName, game.GameWinner.Name))
	}

	// get the amount of wins and idol with most wins for each user
	userRankings := []*rankingStruct{}
	for userId, gameWinners := range rankingsInfo {
		userRankingInfo := &rankingStruct{
			userId:        userId,
			amountOfGames: len(gameWinners),
		}

		// get idol with most wins for this user
		idolCountMap := make(map[string]int)
		highestWins := 0
		for _, idol := range gameWinners {
			idolCountMap[idol]++
		}
		for idol, amountOfGames := range idolCountMap {
			if amountOfGames > highestWins {
				highestWins = amountOfGames
				userRankingInfo.idolWithMostWins = idol
			}
		}

		userRankings = append(userRankings, userRankingInfo)
	}

	// sort rankings by most wins and get top 50
	// sort fields by group name
	sort.Slice(userRankings, func(i, j int) bool {
		return userRankings[i].amountOfGames > userRankings[j].amountOfGames
	})

	if len(userRankings) > 50 {
		userRankings = userRankings[:50]
	}

	embed := &discordgo.MessageEmbed{
		Color: 0x0FADED, // blueish
		Author: &discordgo.MessageEmbedAuthor{
			Name: "Bias Game User Rankings",
		},
	}

	// TODO: this should be updated to use robyuls helpers.GetUser

	// make fields for each group and the idols in the group.
	for i, userRankingInfo := range userRankings {

		userName := "*Unknown User*"
		user, err := cache.GetDiscordSession().User(userRankingInfo.userId)
		if err == nil {
			userName = user.Username
		}

		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   fmt.Sprintf("Rank #%d", i+1),
			Value:  userName,
			Inline: true,
		})
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "Total Games",
			Value:  fmt.Sprintf("%d", userRankingInfo.amountOfGames),
			Inline: true,
		})
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "Most Winning Idol",
			Value:  userRankingInfo.idolWithMostWins,
			Inline: true,
		})
	}

	utils.SendPagedMessage(msg, embed, 21)
}

// displayCurrentGameStats will list the rounds and round winners of a currently running game
func displayCurrentGameStats(msg *discordgo.Message) {

	blankField := &discordgo.MessageEmbedField{
		Name:   ZERO_WIDTH_SPACE,
		Value:  ZERO_WIDTH_SPACE,
		Inline: true,
	}

	// find currently running game for the user or a mention if one exists
	userPlayingGame := msg.Author
	if len(msg.Mentions) > 0 {
		userPlayingGame = msg.Mentions[0]
	}

	if game, ok := currentSinglePlayerGames[userPlayingGame.ID]; ok {

		embed := &discordgo.MessageEmbed{
			Color: 0x0FADED, // blueish
			Author: &discordgo.MessageEmbedAuthor{
				Name: fmt.Sprintf("%s - Current Game Info\n", userPlayingGame.Username),
			},
		}

		// for i := 0; i < len(game.roundWinners); i++ {
		for i := len(game.roundWinners) - 1; i >= 0; i-- {

			fieldName := fmt.Sprintf("Round %d:", i+1)
			if len(game.roundWinners) == i+1 {
				fieldName = "Last Round:"
			}

			message := fmt.Sprintf("W: %s %s\nL: %s %s\n",
				game.roundWinners[i].groupName,
				game.roundWinners[i].biasName,
				game.roundLosers[i].groupName,
				game.roundLosers[i].biasName)

			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
				Name:   fieldName,
				Value:  message,
				Inline: true,
			})
		}

		// notify user if no rounds have been played in the game yet
		if len(embed.Fields) == 0 {
			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
				Name:   "No Rounds",
				Value:  utils.Geti18nText("biasgame.current.no-rounds-played"),
				Inline: true,
			})
		}

		// this is to correct embed alignment
		if len(embed.Fields)%3 == 1 {
			embed.Fields = append(embed.Fields, blankField)
			embed.Fields = append(embed.Fields, blankField)
		} else if len(embed.Fields)%3 == 2 {
			embed.Fields = append(embed.Fields, blankField)
		}

		utils.SendPagedMessage(msg, embed, 12)
	} else {
		utils.SendMessage(msg.ChannelID, "biasgame.current.no-running-game")
	}
}

// recordSingleGamesStats will record the winner, round winners/losers, and other misc stats of a game
func recordSingleGamesStats(game *singleBiasGame) {

	// get guildID from game channel
	channel, _ := cache.GetDiscordSession().State.Channel(game.channelID)
	guild, err := cache.GetDiscordSession().State.Guild(channel.GuildID)
	if err != nil {
		fmt.Println("Error getting guild when recording stats")
		return
	}

	// create a bias game entry
	biasGameEntry := &models.BiasGameEntry{
		ID:           "",
		UserID:       game.user.ID,
		GuildID:      guild.ID,
		GameType:     "single",
		Gender:       game.gender,
		RoundWinners: compileGameWinnersLosers(game.roundWinners),
		RoundLosers:  compileGameWinnersLosers(game.roundLosers),
		GameWinner: models.BiasEntry{
			Name:      game.gameWinnerBias.biasName,
			GroupName: game.gameWinnerBias.groupName,
			Gender:    game.gameWinnerBias.gender,
		},
	}

	utils.MongoDBInsert(models.BiasGameTable, biasGameEntry)
}

// recordSingleGamesStats will record the winner, round winners/losers, and other misc stats of a game
func recordMultiGamesStats(game *multiBiasGame) {

	// get guildID from game channel
	channel, _ := cache.GetDiscordSession().State.Channel(game.channelID)
	guild, err := cache.GetDiscordSession().State.Guild(channel.GuildID)
	if err != nil {
		fmt.Println("Error getting guild when recording stats")
		return
	}

	// create a bias game entry
	biasGameEntry := &models.BiasGameEntry{
		ID:           "",
		GuildID:      guild.ID,
		GameType:     "multi",
		Gender:       game.gender,
		RoundWinners: compileGameWinnersLosers(game.roundWinners),
		RoundLosers:  compileGameWinnersLosers(game.roundLosers),
		GameWinner: models.BiasEntry{
			Name:      game.gameWinnerBias.biasName,
			GroupName: game.gameWinnerBias.groupName,
			Gender:    game.gameWinnerBias.gender,
		},
	}

	utils.MongoDBInsert(models.BiasGameTable, biasGameEntry)
}

// getStatsResults will get the stats results based on the stats message
func getStatsResults(msg *discordgo.Message, statsMessage string) (*mgo.Query, string, string) {
	iconURL := ""
	targetName := ""
	guild, err := utils.GetGuildFromMessage(msg)

	queryParams := bson.M{}

	// filter by game type. multi/single
	if strings.Contains(statsMessage, "multi") {
		queryParams["gametype"] = "multi"

		// multi stats games can run for server or global with server as the default
		if strings.Contains(statsMessage, "global") {

			iconURL = cache.GetDiscordSession().State.User.AvatarURL("512")
			targetName = "Global"
		} else {
			queryParams["guildid"] = guild.ID
			iconURL = discordgo.EndpointGuildIcon(guild.ID, guild.Icon)
			targetName = "Server"

		}
	} else {
		queryParams["gametype"] = "single"

		// user/server/global checks
		if strings.Contains(statsMessage, "server") {

			if err != nil {
				// todo: a message here or something i guess?
			}

			iconURL = discordgo.EndpointGuildIcon(guild.ID, guild.Icon)
			targetName = "Server"
			queryParams["guildid"] = guild.ID
		} else if strings.Contains(statsMessage, "global") {
			iconURL = cache.GetDiscordSession().State.User.AvatarURL("512")
			targetName = "Global"

		} else if strings.Contains(statsMessage, "@") {
			iconURL = msg.Mentions[0].AvatarURL("512")
			targetName = msg.Mentions[0].Username

			queryParams["userid"] = msg.Mentions[0].ID
		} else {
			iconURL = msg.Author.AvatarURL("512")
			targetName = msg.Author.Username

			queryParams["userid"] = msg.Author.ID
		}

	}

	// filter by gamewinner gender
	if strings.Contains(statsMessage, "boy") || strings.Contains(statsMessage, "boys") {
		queryParams["gamewinner.gender"] = "boy"
	} else if strings.Contains(statsMessage, "girl") || strings.Contains(statsMessage, "girls") {
		queryParams["gamewinner.gender"] = "girl"
	}

	//  Note: not sure if want to do dates. might be kinda cool. but could cause confusion due to timezone issues
	// date checks
	// if strings.Contains(statsMessage, "today") {
	// 	// dateCheck := bson.NewObjectIdWithTime()
	// 	messageTime, _ := msg.Timestamp.Parse()

	// 	from := time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 0, 0, 0, 0, messageTime.Location())
	// 	to := time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 23, 59, 59, 0, messageTime.Location())

	// 	fromId := bson.NewObjectIdWithTime(from)
	// 	toId := bson.NewObjectIdWithTime(to)

	// 	queryParams["_id"] = bson.M{"$gte": fromId, "$lt": toId}
	// }

	return utils.MongoDBSearch(models.BiasGameTable, queryParams), iconURL, targetName
}

// complieGameStats will convert records from database into a:
// 		map[int number of occurentces]string group or biasnames comma delimited
// 		will also return []int of the sorted unique counts for reliable looping later
func complieGameStats(records map[string]int) (map[int][]string, []int) {

	// use map of counts to compile a new map of [unique occurence amounts]biasnames
	var uniqueCounts []int
	compiledData := make(map[int][]string)
	for k, v := range records {
		// store unique counts so the map can be "sorted"
		if _, ok := compiledData[v]; !ok {
			uniqueCounts = append(uniqueCounts, v)
		}

		compiledData[v] = append(compiledData[v], k)
	}

	// sort biggest to smallest
	sort.Sort(sort.Reverse(sort.IntSlice(uniqueCounts)))

	return compiledData, uniqueCounts
}

func sendStatsMessage(msg *discordgo.Message, title string, countLabel string, data map[string]int, iconURL string, targetName string) {

	embed := &discordgo.MessageEmbed{
		Color: 0x0FADED, // blueish
		Author: &discordgo.MessageEmbedAuthor{
			Name:    fmt.Sprintf("%s - %s\n", targetName, title),
			IconURL: iconURL,
		},
	}

	// convert data to map[num of occurences]delimited biases
	compiledData, uniqueCounts := complieGameStats(data)
	for _, count := range uniqueCounts {

		// sort biases by group
		sort.Slice(compiledData[count], func(i, j int) bool {
			return compiledData[count][i] < compiledData[count][j]
		})

		joinedNames := strings.Join(compiledData[count], ", ")

		if len(joinedNames) < 1024 {
			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
				Name:   fmt.Sprintf("%s - %d", countLabel, count),
				Value:  joinedNames,
				Inline: false,
			})

		} else {

			// for a specific count, split into multiple fields of at max 40 names
			dataForCount := compiledData[count]
			namesPerField := 40
			breaker := true
			for breaker {

				var namesForField string
				if len(dataForCount) >= namesPerField {
					namesForField = strings.Join(dataForCount[:namesPerField], ", ")
					dataForCount = dataForCount[namesPerField:]
				} else {
					namesForField = strings.Join(dataForCount, ", ")
					breaker = false
				}

				embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
					Name:   fmt.Sprintf("%s - %d", countLabel, count),
					Value:  namesForField,
					Inline: false,
				})

			}
		}

	}

	// send paged message with 5 fields per page
	utils.SendPagedMessage(msg, embed, 5)
}

// compileGameWinnersLosers will loop through the biases and convert them to []models.BiasEntry
func compileGameWinnersLosers(biases []*biasChoice) []models.BiasEntry {

	var biasEntries []models.BiasEntry
	for _, bias := range biases {
		biasEntries = append(biasEntries, models.BiasEntry{
			Name:      bias.biasName,
			GroupName: bias.groupName,
			Gender:    bias.gender,
		})
	}

	return biasEntries
}
