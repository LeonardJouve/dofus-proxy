package cosmetic

import (
	"dofus-proxy/proto/game/common"
	"dofus-proxy/proto/game/cosmetic"
	"dofus-proxy/proxy"
	"errors"
	"fmt"
	"time"

	game "dofus-proxy/proto/game/message"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

const (
	COSMETIC_INVENTORY_MODIFIER     = "cosmetic_inventory_modifier"
	COSMETIC_INVENTORY_EQUIP_FILTER = "cosmetic_inventory_equip_filter"
)

func AddCosmeticInventoryModifier(p *proxy.Proxy) string {
	p.AddModifier(COSMETIC_INVENTORY_MODIFIER, func(message proto.Message) (proto.Message, error) {
		gameMessage, ok := message.(*game.Message)
		if !ok {
			return nil, errors.New("not a game message")
		}

		if _, ok := gameMessage.Content.(*game.Message_Event); !ok {
			return nil, errors.New("not a game event")
		}

		event := gameMessage.GetEvent()
		if event.GetContent().GetTypeUrl() != "type.ankama.com/jue" {
			return nil, errors.New("not a cosmetic inventory loaded event")
		}

		value, err := proto.Marshal(&cosmetic.CosmeticInventoryLoadedEvent{
			Objects: []*common.ObjectItem{
				{
					Uid:      1,
					Quantity: 1,
					Gid:      26243,
				},
			},
		})
		if err != nil {
			return nil, err
		}

		event.GetContent().Value = value

		return gameMessage, nil
	})

	return COSMETIC_INVENTORY_MODIFIER
}

func AddCosmeticInventoryEquipFilter(p *proxy.Proxy) string {
	p.AddFilter(COSMETIC_INVENTORY_EQUIP_FILTER, func(message proto.Message) bool {
		gameMessage, ok := message.(*game.Message)
		if !ok {
			return false
		}

		if _, ok := gameMessage.Content.(*game.Message_Request); !ok {
			return false
		}

		request := gameMessage.GetRequest()

		if request.GetContent().GetTypeUrl() != "type.ankama.com/jut" {
			return false
		}

		fmt.Println("Intercept equip")

		uid := request.GetUid()
		go func() {
			time.Sleep(200 * time.Millisecond)

			outfitEquipResponse, err := proto.Marshal(&cosmetic.OutfitEquipResponse{
				Success: true,
			})
			if err != nil {
				return
			}

			response := game.Message{
				Content: &game.Message_Response{
					Response: &game.Response{
						Uid: uid,
						Content: &anypb.Any{
							TypeUrl: "type.ankama.com/juu",
							Value:   outfitEquipResponse,
						},
					},
				},
			}

			fmt.Println("Inject accept")

			p.InjectServer(&response)
		}()

		return true
	})
	return COSMETIC_INVENTORY_EQUIP_FILTER
}
