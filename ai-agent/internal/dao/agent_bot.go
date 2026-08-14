package dao

import (
	"context"

	"github.com/gly-hub/ai-dandelion/ai-agent/internal/model"
	"gorm.io/gorm"
)

type AgentBot struct {
	db *gorm.DB
}

type AgentBotAggregate struct {
	Bot          model.AgentBot
	Channels     []model.AgentBotChannel
	Capabilities []model.AgentBotCapability
}

func NewAgentBot(db *gorm.DB) *AgentBot {
	return &AgentBot{db: db}
}

func (d *AgentBot) List(ctx context.Context) ([]AgentBotAggregate, error) {
	var bots []model.AgentBot
	if err := d.db.WithContext(ctx).Order("updated_at DESC, created_at DESC").Find(&bots).Error; err != nil {
		return nil, err
	}
	out := make([]AgentBotAggregate, 0, len(bots))
	for i := range bots {
		item, err := d.loadAggregate(ctx, bots[i])
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, nil
}

func (d *AgentBot) Get(ctx context.Context, id string) (*AgentBotAggregate, error) {
	var bot model.AgentBot
	if err := d.db.WithContext(ctx).Where("id = ?", id).First(&bot).Error; err != nil {
		return nil, err
	}
	item, err := d.loadAggregate(ctx, bot)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (d *AgentBot) Create(ctx context.Context, bot *model.AgentBot, channels []model.AgentBotChannel, capabilities []model.AgentBotCapability) error {
	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(bot).Error; err != nil {
			return err
		}
		if len(channels) > 0 {
			if err := tx.Create(&channels).Error; err != nil {
				return err
			}
		}
		if len(capabilities) > 0 {
			if err := tx.Create(&capabilities).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (d *AgentBot) Update(ctx context.Context, bot *model.AgentBot, channels []model.AgentBotChannel, capabilities []model.AgentBotCapability) error {
	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(bot).Error; err != nil {
			return err
		}
		if err := tx.Where("bot_id = ?", bot.ID).Delete(&model.AgentBotChannel{}).Error; err != nil {
			return err
		}
		if err := tx.Where("bot_id = ?", bot.ID).Delete(&model.AgentBotCapability{}).Error; err != nil {
			return err
		}
		if len(channels) > 0 {
			if err := tx.Create(&channels).Error; err != nil {
				return err
			}
		}
		if len(capabilities) > 0 {
			if err := tx.Create(&capabilities).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (d *AgentBot) Delete(ctx context.Context, id string) error {
	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("bot_id = ?", id).Delete(&model.AgentBotChannel{}).Error; err != nil {
			return err
		}
		if err := tx.Where("bot_id = ?", id).Delete(&model.AgentBotCapability{}).Error; err != nil {
			return err
		}
		result := tx.Where("id = ?", id).Delete(&model.AgentBot{})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return nil
	})
}

func (d *AgentBot) SaveBot(ctx context.Context, bot *model.AgentBot) error {
	return d.db.WithContext(ctx).Save(bot).Error
}

func (d *AgentBot) loadAggregate(ctx context.Context, bot model.AgentBot) (AgentBotAggregate, error) {
	var channels []model.AgentBotChannel
	if err := d.db.WithContext(ctx).Where("bot_id = ?", bot.ID).Order("created_at ASC").Find(&channels).Error; err != nil {
		return AgentBotAggregate{}, err
	}
	var capabilities []model.AgentBotCapability
	if err := d.db.WithContext(ctx).Where("bot_id = ?", bot.ID).Order("capability_type ASC, created_at ASC").Find(&capabilities).Error; err != nil {
		return AgentBotAggregate{}, err
	}
	return AgentBotAggregate{
		Bot:          bot,
		Channels:     channels,
		Capabilities: capabilities,
	}, nil
}
