package svc

import (
	"github.com/apache/pulsar-client-go/pulsar"
	"github.com/luxun9527/gex/app/quotes/kline/rpc/internal/config"
	"github.com/luxun9527/gex/app/quotes/kline/rpc/internal/dao/query"
	pulsarConfig "github.com/luxun9527/gex/common/pkg/pulsar"
	"github.com/luxun9527/gex/common/proto/define"
	gpushPb "github.com/luxun9527/gpush/proto"
	logger "github.com/luxun9527/zaplog"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/zrpc"
)

type ServiceContext struct {
	Config        *config.Config
	Query         *query.Query
	RedisClient   *redis.Redis
	MatchConsumer pulsar.Consumer
	WsClient      gpushPb.ProxyClient
}

func NewServiceContext(c *config.Config) *ServiceContext {
	logger.InitZapLogger(&c.LoggerConfig)
	logx.SetWriter(logger.NewZapWriter(logger.GetZapLogger()))
	logx.DisableStat()

	var symbolInfo define.SymbolInfo
	define.InitSymbolConfig(define.EtcdSymbolPrefix+c.Symbol, c.SymbolEtcdConfig, &symbolInfo)
	c.SymbolInfo = &symbolInfo
	logx.Infow("symbol config load ", logx.Field("symbol", symbolInfo))

	c.Etcd.Key += "." + c.Symbol

	client, err := c.PulsarConfig.BuildClient()
	if err != nil {
		logx.Severef("init pulsar client failed %v", err)
	}
	topic := pulsarConfig.Topic{
		Tenant:    pulsarConfig.PublicTenant,
		Namespace: pulsarConfig.GexNamespace,
		Topic:     pulsarConfig.MatchResultTopic + "_" + c.Symbol,
	}
	consumer, err := client.Subscribe(pulsar.ConsumerOptions{
		Topic:            topic.BuildTopic(),
		SubscriptionName: pulsarConfig.MatchResultKlineSub,
		Type:             pulsar.Shared,
	})
	if err != nil {
		logx.Severef("init pulsar consumer failed %v", err)
	}
	sc := &ServiceContext{
		Config:        c,
		Query:         query.Use(c.GormConf.MustNewGormClient()),
		RedisClient:   redis.MustNewRedis(c.RedisConf),
		MatchConsumer: consumer,
		WsClient:      gpushPb.NewProxyClient(zrpc.MustNewClient(c.WsConf).Conn()),
	}
	return sc
}
