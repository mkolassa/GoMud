package usercommands

import (
	"fmt"

	"github.com/volte6/mud/buffs"
	"github.com/volte6/mud/items"
	"github.com/volte6/mud/rooms"
	"github.com/volte6/mud/users"
	"github.com/volte6/mud/util"
)

func Eat(rest string, userId int, cmdQueue util.CommandQueue) (util.MessageQueue, error) {

	response := NewUserCommandResponse(userId)

	// Load user details
	user := users.GetByUserId(userId)
	if user == nil { // Something went wrong. User not found.
		return response, fmt.Errorf("user %d not found", userId)
	}

	// Load current room details
	room := rooms.LoadRoom(user.Character.RoomId)
	if room == nil {
		return response, fmt.Errorf(`room %d not found`, user.Character.RoomId)
	}

	// Check whether the user has an item in their inventory that matches
	matchItem, found := user.Character.FindInBackpack(rest)

	if !found {
		response.SendUserMessage(userId, fmt.Sprintf(`You don't have a "%s" to eat.`, rest), true)
	} else {

		itemSpec := matchItem.GetSpec()

		if itemSpec.Subtype != items.Edible {
			response.SendUserMessage(userId,
				fmt.Sprintf(`You can't eat <ansi fg="itemname">%s</ansi>.`, matchItem.Name()),
				true)
			response.Handled = true
			return response, nil
		}

		user.Character.CancelBuffsWithFlag(buffs.Hidden)

		usesBefore := matchItem.Uses
		user.Character.UseItem(matchItem)
		usesAfter := matchItem.Uses
		fmt.Println("Uses", usesBefore, usesAfter)
		response.SendUserMessage(userId, fmt.Sprintf(`You eat some of the <ansi fg="itemname">%s</ansi>.`, matchItem.Name()), true)
		response.SendRoomMessage(room.RoomId, fmt.Sprintf(`<ansi fg="username">%s</ansi> eats some <ansi fg="itemname">%s</ansi>.`, user.Character.Name, matchItem.Name()), true)

		for _, buffId := range itemSpec.BuffIds {
			cmdQueue.QueueBuff(user.UserId, 0, buffId)
		}
	}

	response.Handled = true
	return response, nil
}