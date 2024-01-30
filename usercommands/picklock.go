package usercommands

import (
	"fmt"
	"strings"

	"github.com/volte6/mud/configs"
	"github.com/volte6/mud/items"
	"github.com/volte6/mud/rooms"
	"github.com/volte6/mud/templates"
	"github.com/volte6/mud/users"
	"github.com/volte6/mud/util"
)

func Picklock(rest string, userId int, cmdQueue util.CommandQueue) (util.MessageQueue, error) {

	response := NewUserCommandResponse(userId)

	// Load user details
	user := users.GetByUserId(userId)
	if user == nil { // Something went wrong. User not found.
		return response, fmt.Errorf("user %d not found", userId)
	}

	lockpickItm := items.Item{}
	for _, itm := range user.Character.GetAllBackpackItems() {
		if itm.GetSpec().Type == items.Lockpicks {
			lockpickItm = itm
			break
		}
	}

	if lockpickItm.ItemId < 1 {
		response.SendUserMessage(userId, `You need <ansi fg="item">lockpicks</ansi> to pick a lock.`, true)
		response.Handled = true
		return response, nil
	}

	// Load current room details
	room := rooms.LoadRoom(user.Character.RoomId)
	if room == nil {
		return response, fmt.Errorf(`room %d not found`, user.Character.RoomId)
	}

	args := util.SplitButRespectQuotes(strings.ToLower(rest))

	if len(args) < 1 {
		response.SendUserMessage(userId, "You wanna pock a lock? Specify where it is.", true)
		response.Handled = true
		return response, nil
	}

	lockId := ``
	lockStrength := 0

	containerName := room.FindContainerByName(args[0])
	exitName, exitRoomId := room.FindExitByName(args[0])

	if containerName != `` {

		container := room.Containers[containerName]

		if !container.HasLock() {
			response.SendUserMessage(userId, "There is no lock there.", true)
			response.Handled = true
			return response, nil
		}

		if !container.Lock.IsLocked() {
			response.SendUserMessage(userId, "It's already unlocked.", true)
			response.Handled = true
			return response, nil
		}

		args = args[1:]
		lockStrength = int(container.Lock.Difficulty)
		lockId = fmt.Sprintf(`%d-%s`, room.RoomId, containerName)

	} else if exitRoomId > 0 {

		// get the first entry int he slice and shorten the slice
		args = args[1:]

		exitInfo := room.Exits[exitName]

		if !exitInfo.HasLock() {
			response.SendUserMessage(userId, "There is no lock there.", true)
			response.Handled = true
			return response, nil
		}

		if !exitInfo.Lock.IsLocked() {
			response.SendUserMessage(userId, "It's already unlocked.", true)
			response.Handled = true
			return response, nil
		}

		lockStrength = int(exitInfo.Lock.Difficulty)
		lockId = fmt.Sprintf(`%d-%s`, room.RoomId, exitName)

	} else {

		response.SendUserMessage(userId, "There is no such exit or container.", true)
		response.Handled = true
		return response, nil
	}

	//
	// Most of what follows shouldn't reference an exit or a chest, but rather lock details.
	//
	rows := [][]string{}

	keyring_sequence := user.Character.GetKey(lockId)

	sequence := util.GetLockSequence(lockId, lockStrength, configs.GetConfig().Seed)

	inspect_only := false

	if keyring_sequence == sequence {
		response.SendUserMessage(userId, "", true)
		response.SendUserMessage(userId, "Your keyring already has this lock on it.", true)
		inspect_only = true
	}

	entered := ``

	if len(keyring_sequence) > 0 {
		entered = keyring_sequence
	}

	if len(args) == 0 {

		inspect_only = true

	} else if !inspect_only {

		for _, r := range args {

			done := false
			r := strings.ToUpper(r)
			r = string(r[0])

			if r != "U" && r != "D" {
				done = true
				break
			}

			entered += r

			if done {
				break
			}
		}

		for i := 0; i < len(entered); i++ {
			if entered[i] != sequence[i] {
				// Mismatch! BREAKS!
				entered = ``
				user.Character.UseItem(lockpickItm)

				response.SendUserMessage(userId, ``, true)
				response.SendUserMessage(userId, fmt.Sprintf(`<ansi fg="yellow" bold="true">***</ansi> <ansi fg="red" bold="true">Oops! Your <ansi fg="item">%s</ansi> break off in the lock, resetting the lock. You'll have to start all over.</ansi> <ansi fg="yellow" bold="true">***</ansi>`, lockpickItm.GetSpec().NameSimple), true)
			}
		}

		user.Character.SetKey(lockId, entered)

		if len(entered) > 0 {
			response.SendUserMessage(userId, ``, true)
			response.SendUserMessage(userId, `<ansi fg="green" bold="true">A satisfying *click* tells you that you're making progress...</ansi>`, true)
		}

	}

	if len(entered) > len(sequence) {
		entered = entered[:len(sequence)]
	}

	formatting := make([]string, len(sequence))

	row := []string{}
	for i := 0; i < len(sequence); i++ {
		if len(entered) > i && entered[i] == sequence[i] && entered[i] == 'U' {
			row = append(row, `  U  `)
			formatting[i] = `<ansi fg="green" bold="true">%s</ansi>`
		} else {
			row = append(row, `     `)
		}
	}
	rows = append(rows, row)

	row = []string{}
	for i := 0; i < len(sequence); i++ {
		if i >= len(entered) || entered[i] != sequence[i] {
			row = append(row, `  ?  `)
			formatting[i] = `<ansi fg="red" bold="true">%s</ansi>`
		} else {
			if entered[i] == 'U' {
				row = append(row, `  ↑  `)
			} else if entered[i] == 'D' {
				row = append(row, `  ↓  `)
			} else {
				row = append(row, `     `)
			}
		}
	}
	rows = append(rows, row)

	row = []string{}
	for i := 0; i < len(sequence); i++ {
		if len(entered) > i && entered[i] == sequence[i] && entered[i] == 'D' {
			row = append(row, `  D  `)
			formatting[i] = `<ansi fg="green" bold="true">%s</ansi>`
		} else {
			row = append(row, `     `)
		}
	}
	rows = append(rows, row)

	picklockTable := templates.GetTable(`The Lock Sequence Looks like:`, rows[0], rows, formatting)
	tplTxt, _ := templates.Process("tables/lockpicking", picklockTable)
	response.SendUserMessage(userId, tplTxt, false)

	if sequence == entered {

		response.SendUserMessage(userId, `<ansi fg="yellow" bold="true">***</ansi> <ansi fg="green" bold="true">You Successfully picked the lock!</ansi> <ansi fg="yellow" bold="true">***</ansi>`, true)
		response.SendUserMessage(userId, `<ansi fg="yellow" bold="true">***</ansi> <ansi fg="green" bold="true">You can automatically pick this lock any time as long as you carry <ansi fg="item">lockpicks</ansi>!</ansi> <ansi fg="yellow" bold="true">***</ansi>`, true)
		response.SendUserMessage(userId, ``, true)

		if containerName != `` {

			response.SendRoomMessage(user.Character.RoomId, fmt.Sprintf(`<ansi fg="username">%s</ansi> picks the <ansi fg="container">%s</ansi> lock`, user.Character.Name, containerName), true)

			container := room.Containers[containerName]
			container.Lock.SetUnlocked()
			room.Containers[containerName] = container
		} else {

			response.SendRoomMessage(user.Character.RoomId, fmt.Sprintf(`<ansi fg="username">%s</ansi> picks the <ansi fg="exit">%s</ansi> lock`, user.Character.Name, exitName), true)

			exitInfo := room.Exits[exitName]
			exitInfo.Lock.SetUnlocked()
			room.Exits[exitName] = exitInfo
		}
	} else {
		if containerName != `` {
			response.SendRoomMessage(user.Character.RoomId, fmt.Sprintf(`<ansi fg="username">%s</ansi> tries to pick the <ansi fg="container">%s</ansi> lock`, user.Character.Name, containerName), true)
		} else {
			response.SendRoomMessage(user.Character.RoomId, fmt.Sprintf(`<ansi fg="username">%s</ansi> tries to pick the <ansi fg="exit">%s</ansi> lock`, user.Character.Name, exitName), true)
		}
	}

	response.Handled = true
	return response, nil
}