package mill

import (
	"github.com/IBM/sarama"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/wfusion/gofusion/common/utils"

	"github.com/wfusion/gofusion/common/infra/watermill/pubsub/kafka"
)

var kafkaCmd = &cobra.Command{
	Use:   "kafka",
	Short: "Commands for the kafka Pub/Sub provider",
	Long: `Consume or produce messages from the kafka Pub/Sub provider.

For the configuration of consuming/producing of the messages, check the help of the relevant command.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		err := rootCmd.PersistentPreRunE(cmd, args)
		if err != nil {
			return err
		}
		logger.Debug("Using kafka pub/sub", nil)

		brokers := viper.GetStringSlice("kafka.brokers")

		if cmd.Use == "consume" {
			saramaSubscriberConfig := kafka.DefaultSaramaSubscriberConfig()

			if viper.GetBool("kafka.consume.fromBeginning") {
				logger.Trace("Configured sarama to consume messages from beginning", nil)
				// equivalent of auto.offset.reset: earliest
				saramaSubscriberConfig.Consumer.Offsets.Initial = sarama.OffsetOldest
			}

			consumer, err = kafka.NewSubscriber(
				kafka.SubscriberConfig{
					Brokers:               brokers,
					Unmarshaler:           kafka.DefaultMarshaler{},
					OverwriteSaramaConfig: saramaSubscriberConfig,
					ConsumerGroup:         viper.GetString("kafka.consume.consumerGroup"),
				},
				logger,
			)
			if err != nil {
				return err
			}
		}

		if cmd.Use == "produce" {
			producer, err = kafka.NewPublisher(
				kafka.PublisherConfig{
					Brokers:   brokers,
					Marshaler: kafka.DefaultMarshaler{},
				},
				logger,
			)
			if err != nil {
				return err
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(kafkaCmd)

	kafkaCmd.PersistentFlags().StringP(
		"topic",
		"t",
		"",
		"The topic to produce messages to (produce) or consume message from (consume)",
	)
	utils.MustSuccess(kafkaCmd.MarkPersistentFlagRequired("topic"))
	utils.MustSuccess(viper.BindPFlag("kafka.topic", kafkaCmd.PersistentFlags().Lookup("topic")))

	consumeCmd := addConsumeCmd(kafkaCmd, "kafka.topic")
	_ = addProduceCmd(kafkaCmd, "kafka.topic")

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	kafkaCmd.PersistentFlags().StringSliceP("brokers", "b", nil, "A list of kafka brokers")
	utils.MustSuccess(kafkaCmd.MarkPersistentFlagRequired("brokers"))
	utils.MustSuccess(viper.BindPFlag("kafka.brokers", kafkaCmd.PersistentFlags().Lookup("brokers")))

	consumeCmd.PersistentFlags().Bool("from-beginning", false, "Equivalent to auto.offset.reset: earliest")
	utils.MustSuccess(viper.BindPFlag("kafka.consume.fromBeginning", consumeCmd.PersistentFlags().Lookup("from-beginning")))

	consumeCmd.PersistentFlags().StringP("consumer-group", "c", "", "The kafka consumer group. Defaults to empty.")
	utils.MustSuccess(viper.BindPFlag("kafka.consume.consumerGroup", consumeCmd.PersistentFlags().Lookup("consumer-group")))
}
