package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"sync"
	"time"

	expr "cel.dev/expr"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/executor"
	dtktgraph "github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/graph"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/pubsub"
	memorypubsub "github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/pubsub/memory"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/runtime"
	flowv1beta2 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta2"
)

var inputs_number = []int{1, 2, 3, 3, 3, 3, 3, 4, 5, 6, 7, 8, 8, 8, 8, 8, 9, 10}

/* Flow:
# pseudo yaml flow spec
inputs:
- id: number
  type: int64
vars:
- id: evenNumbers
  value: = inputs.number.value
  transforms:
  - filter: = this.value % 2 == 0
- id: oddNumbers
  value: = inputs.number.value
  transforms:
  - filter: = this.value % 2 != 0
outputs:
- id: evenNumbers
  value: = vars.evenNumbers.value
- id: oddNumbers
  value: = vars.oddNumbers.value
- id: evenOddPairs
  value: = [vars.evenNumbers.value, vars.oddNumbers.value]
- id: evenSum
  value: = vars.evenNumbers.value
  transforms:
  - reduce:
      initial: = 0
      accumulator: = this.accumulator + this.value
- id: oddSum
  value: = vars.oddNumbers.value
  transforms:
  - reduce:
      initial: = 0
      accumulator: = this.accumulator + this.value
      group_by:
        window:
          event:
            when: = vars.oddNumbers.closed
*/

func main() {
	numberInput := &flowv1beta2.Input{}
	numberInput.SetId("number")
	numberInput.SetInt64(&flowv1beta2.Int64{})

	evenFilter := &flowv1beta2.Transform{}
	evenFilter.SetFilter("= this.value % 2 == 0")

	oddFilter := &flowv1beta2.Transform{}
	oddFilter.SetFilter("= this.value % 2 != 0")

	evenVar := &flowv1beta2.Var{}
	evenVar.SetId("evenNumbers")
	evenVar.SetValue("= inputs.number.value")
	evenVar.SetTransforms([]*flowv1beta2.Transform{evenFilter})

	oddVar := &flowv1beta2.Var{}
	oddVar.SetId("oddNumbers")
	oddVar.SetValue("= inputs.number.value")
	oddVar.SetTransforms([]*flowv1beta2.Transform{oddFilter})

	evenOut := &flowv1beta2.Output{}
	evenOut.SetId("evenNumbers")
	evenOut.SetValue("= vars.evenNumbers.value")

	oddOut := &flowv1beta2.Output{}
	oddOut.SetId("oddNumbers")
	oddOut.SetValue("= vars.oddNumbers.value")

	pairsOut := &flowv1beta2.Output{}
	pairsOut.SetId("evenOddPairs")
	pairsOut.SetValue("= [vars.evenNumbers.value, vars.oddNumbers.value]")

	evenSumReduce := &flowv1beta2.Transform_Reduce{}
	evenSumReduce.SetInitial("= 0")
	evenSumReduce.SetAccumulator("= this.accumulator + this.value")
	evenSumTransform := &flowv1beta2.Transform{}
	evenSumTransform.SetReduce(evenSumReduce)

	evenSumOut := &flowv1beta2.Output{}
	evenSumOut.SetId("evenSum")
	evenSumOut.SetValue("= vars.evenNumbers.value")
	evenSumOut.SetTransforms([]*flowv1beta2.Transform{evenSumTransform})

	eventWindow := &flowv1beta2.Transform_GroupBy_Window_Event{}
	eventWindow.SetWhen("= vars.oddNumbers.closed")
	gbWindow := &flowv1beta2.Transform_GroupBy_Window{}
	gbWindow.SetEvent(eventWindow)
	groupBy := &flowv1beta2.Transform_GroupBy{}
	groupBy.SetWindow(gbWindow)
	oddSumReduce := &flowv1beta2.Transform_Reduce{}
	oddSumReduce.SetInitial("= 0")
	oddSumReduce.SetAccumulator("= this.accumulator + this.value")
	oddSumReduce.SetGroupBy(groupBy)
	oddSumTransform := &flowv1beta2.Transform{}
	oddSumTransform.SetReduce(oddSumReduce)

	oddSumOut := &flowv1beta2.Output{}
	oddSumOut.SetId("oddSum")
	oddSumOut.SetValue("= vars.oddNumbers.value")
	oddSumOut.SetTransforms([]*flowv1beta2.Transform{oddSumTransform})

	flow := flowv1beta2.Flow{}
	flow.SetName("even-odd-sum")
	flow.SetInputs([]*flowv1beta2.Input{numberInput})
	flow.SetVars([]*flowv1beta2.Var{evenVar, oddVar})
	flow.SetOutputs([]*flowv1beta2.Output{evenOut, oddOut, pairsOut, evenSumOut, oddSumOut})

	// Build graph from flow.
	graph, err := dtktgraph.Build(&flow)
	if err != nil {
		log.Fatal(err)
	}

	// Debug: print DOT representation.
	// if dot, err := dtktgraph.DOT(graph); err == nil {
	// 	fmt.Println(dot)
	// }

	ps := memorypubsub.New(memorypubsub.WithPersistent())
	defer func() {
		if err := ps.Close(); err != nil {
			slog.Error("ps.Close", slog.Any("err", err))
		}
	}()
	topics := executor.NewTopics("example")
	exec := runtime.NewExecutor(ps, topics)

	// Feed input values to PubSub topic.
	inputTopic := topics.InputFor("inputs.number")
	for _, v := range inputs_number {
		val := &expr.Value{Kind: &expr.Value_Int64Value{Int64Value: int64(v)}}
		if err := ps.Publish(inputTopic, pubsub.NewMessage(val)); err != nil {
			log.Fatal(err)
		}
	}
	// Send EOF marker.
	if err := ps.Publish(inputTopic, pubsub.NewMessage(runtime.NewEOFValue())); err != nil {
		log.Fatal(err)
	}

	if err := exec.Execute(context.Background(), graph); err != nil {
		log.Fatal(err)
	}

	// Read outputs from PubSub.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	outputIDs := []string{
		"outputs.evenNumbers",
		"outputs.oddNumbers",
		"outputs.evenOddPairs",
		"outputs.evenSum",
		"outputs.oddSum",
	}

	var wg sync.WaitGroup
	for _, id := range outputIDs {
		ch, _ := ps.Subscribe(ctx, topics.For(id))
		wg.Add(1)
		go func(id string, ch <-chan *pubsub.Message) {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case msg := <-ch:
					evt := msg.Payload.(*flowv1beta2.RunSnapshot_NodeEvent)
					node := evt.GetOutput()
					msg.Ack()
					if node.GetClosed() {
						return
					}
					fmt.Printf("[%s] %v\n", node.GetId(), node.GetValue())
				}
			}
		}(id, ch)
	}
	wg.Wait()
}
