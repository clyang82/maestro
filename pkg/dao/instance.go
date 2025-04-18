package dao

import (
	"context"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm/clause"

	"github.com/openshift-online/maestro/pkg/api"
	"github.com/openshift-online/maestro/pkg/db"
)

type InstanceDao interface {
	Get(ctx context.Context, id string) (*api.ServerInstance, error)
	Create(ctx context.Context, instance *api.ServerInstance) (*api.ServerInstance, error)
	Replace(ctx context.Context, instance *api.ServerInstance) (*api.ServerInstance, error)
	MarkReadyByIDs(ctx context.Context, ids []string) error
	MarkUnreadyByIDs(ctx context.Context, ids []string) error
	Delete(ctx context.Context, id string) error
	DeleteByIDs(ctx context.Context, ids []string) error
	FindByIDs(ctx context.Context, ids []string) (api.ServerInstanceList, error)
	FindByUpdatedTime(ctx context.Context, updatedTime time.Time) (api.ServerInstanceList, error)
	FindReadyIDs(ctx context.Context) ([]string, error)
	All(ctx context.Context) (api.ServerInstanceList, error)
}

var _ InstanceDao = &sqlInstanceDao{}

type sqlInstanceDao struct {
	sessionFactory *db.SessionFactory
}

func NewInstanceDao(sessionFactory *db.SessionFactory) InstanceDao {
	return &sqlInstanceDao{sessionFactory: sessionFactory}
}

func (d *sqlInstanceDao) Get(ctx context.Context, id string) (*api.ServerInstance, error) {
	g2 := (*d.sessionFactory).New(ctx)
	var instance api.ServerInstance
	if err := g2.Take(&instance, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &instance, nil
}

func (d *sqlInstanceDao) Create(ctx context.Context, instance *api.ServerInstance) (*api.ServerInstance, error) {
	g2 := (*d.sessionFactory).New(ctx)
	if err := g2.Omit(clause.Associations).Create(instance).Error; err != nil {
		db.MarkForRollback(ctx, err)
		return nil, err
	}
	return instance, nil
}

func (d *sqlInstanceDao) Replace(ctx context.Context, instance *api.ServerInstance) (*api.ServerInstance, error) {
	g2 := (*d.sessionFactory).New(ctx)
	if err := g2.Omit(clause.Associations).Save(instance).Error; err != nil {
		db.MarkForRollback(ctx, err)
		return nil, err
	}
	return instance, nil
}

func (d *sqlInstanceDao) MarkReadyByIDs(ctx context.Context, ids []string) error {
	g2 := (*d.sessionFactory).New(ctx)
	if err := g2.Model(&api.ServerInstance{}).Where("id in (?)", ids).Update("ready", true).Error; err != nil {
		db.MarkForRollback(ctx, err)
		return err
	}

	// call pg_notify to notify the server_instances channel
	notify := fmt.Sprintf("select pg_notify('%s', '%s')", "server_instances", fmt.Sprintf("ready:%s", strings.Join(ids, ",")))
	return g2.Exec(notify).Error
}

func (d *sqlInstanceDao) MarkUnreadyByIDs(ctx context.Context, ids []string) error {
	g2 := (*d.sessionFactory).New(ctx)
	if err := g2.Model(&api.ServerInstance{}).Where("id in (?)", ids).Update("ready", false).Error; err != nil {
		db.MarkForRollback(ctx, err)
		return err
	}
	// call pg_notify to notify the server_instances channel
	notify := fmt.Sprintf("select pg_notify('%s', '%s')", "server_instances", fmt.Sprintf("unready:%s", strings.Join(ids, ",")))
	return g2.Exec(notify).Error
}

func (d *sqlInstanceDao) Delete(ctx context.Context, id string) error {
	g2 := (*d.sessionFactory).New(ctx)
	if err := g2.Omit(clause.Associations).Delete(&api.ServerInstance{Meta: api.Meta{ID: id}}).Error; err != nil {
		db.MarkForRollback(ctx, err)
		return err
	}
	return nil
}

func (d *sqlInstanceDao) DeleteByIDs(ctx context.Context, ids []string) error {
	g2 := (*d.sessionFactory).New(ctx)
	instances := api.ServerInstanceList{}
	for _, id := range ids {
		instances = append(instances, &api.ServerInstance{Meta: api.Meta{ID: id}})
	}
	if err := g2.Omit(clause.Associations).Delete(&instances).Error; err != nil {
		db.MarkForRollback(ctx, err)
		return err
	}

	return nil
}

func (d *sqlInstanceDao) FindByIDs(ctx context.Context, ids []string) (api.ServerInstanceList, error) {
	g2 := (*d.sessionFactory).New(ctx)
	instances := api.ServerInstanceList{}
	if err := g2.Where("id in (?)", ids).Find(&instances).Error; err != nil {
		return nil, err
	}
	return instances, nil
}

func (d *sqlInstanceDao) FindByUpdatedTime(ctx context.Context, updatedTime time.Time) (api.ServerInstanceList, error) {
	g2 := (*d.sessionFactory).New(ctx)
	instances := api.ServerInstanceList{}
	if err := g2.Where("updated_at < ?", updatedTime).Find(&instances).Error; err != nil {
		return nil, err
	}
	return instances, nil
}

func (d *sqlInstanceDao) All(ctx context.Context) (api.ServerInstanceList, error) {
	g2 := (*d.sessionFactory).New(ctx)
	instances := api.ServerInstanceList{}
	if err := g2.Find(&instances).Error; err != nil {
		return nil, err
	}
	return instances, nil
}

func (d *sqlInstanceDao) FindReadyIDs(ctx context.Context) ([]string, error) {
	g2 := (*d.sessionFactory).New(ctx)
	instances := api.ServerInstanceList{}
	if err := g2.Where("ready = ?", true).Find(&instances).Error; err != nil {
		return nil, err
	}
	ids := make([]string, len(instances))
	for i, instance := range instances {
		ids[i] = instance.ID
	}
	return ids, nil
}
