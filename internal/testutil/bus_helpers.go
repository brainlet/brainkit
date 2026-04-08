package testutil

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/brainlet/brainkit/sdk"
)

// Deploy deploys .ts code via the kit.deploy bus command and waits for the response.
// Fails the test on error or timeout.
func Deploy(t *testing.T, rt sdk.Runtime, source, code string) {
	t.Helper()
	err := DeployErr(rt, source, code)
	if err != nil {
		t.Fatalf("Deploy(%s): %v", source, err)
	}
}

// DeployErr deploys .ts code via the kit.deploy bus command and returns any error.
func DeployErr(rt sdk.Runtime, source, code string) error {
	return DeployWithOpts(rt, source, code, "", "")
}

// DeployWithOpts deploys with optional RBAC role and package name.
func DeployWithOpts(rt sdk.Runtime, source, code, role, packageName string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	pr, err := sdk.PublishKitDeploy(rt, ctx, sdk.KitDeployMsg{
		Source:      source,
		Code:        code,
		Role:        role,
		PackageName: packageName,
	})
	if err != nil {
		return fmt.Errorf("publish kit.deploy: %w", err)
	}

	ch := make(chan error, 1)
	unsub, err := sdk.SubscribeKitDeployResp(rt, ctx, pr.ReplyTo,
		func(resp sdk.KitDeployResp, _ sdk.Message) {
			if resp.Error != "" {
				ch <- fmt.Errorf("%s: %s", resp.Code, resp.Error)
			} else {
				ch <- nil
			}
		})
	if err != nil {
		return fmt.Errorf("subscribe kit.deploy reply: %w", err)
	}
	defer unsub()

	select {
	case err := <-ch:
		return err
	case <-ctx.Done():
		return fmt.Errorf("deploy %s: %w", source, ctx.Err())
	}
}

// DeployWithResources deploys and returns the resource list.
func DeployWithResources(t *testing.T, rt sdk.Runtime, source, code string) []sdk.ResourceInfo {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	pr, err := sdk.PublishKitDeploy(rt, ctx, sdk.KitDeployMsg{Source: source, Code: code})
	if err != nil {
		t.Fatalf("Deploy(%s): publish: %v", source, err)
	}

	type result struct {
		resources []sdk.ResourceInfo
		err       error
	}
	ch := make(chan result, 1)
	unsub, err := sdk.SubscribeKitDeployResp(rt, ctx, pr.ReplyTo,
		func(resp sdk.KitDeployResp, _ sdk.Message) {
			if resp.Error != "" {
				ch <- result{err: fmt.Errorf("%s: %s", resp.Code, resp.Error)}
			} else {
				ch <- result{resources: resp.Resources}
			}
		})
	if err != nil {
		t.Fatalf("Deploy(%s): subscribe: %v", source, err)
	}
	defer unsub()

	select {
	case r := <-ch:
		if r.err != nil {
			t.Fatalf("Deploy(%s): %v", source, r.err)
		}
		return r.resources
	case <-ctx.Done():
		t.Fatalf("Deploy(%s): timeout", source)
		return nil
	}
}

// EvalTS evaluates TypeScript code in the current runtime context via kit.eval-ts bus command.
func EvalTS(t *testing.T, rt sdk.Runtime, source, code string) string {
	t.Helper()
	result, err := EvalTSErr(rt, source, code)
	if err != nil {
		t.Fatalf("EvalTS(%s): %v", source, err)
	}
	return result
}

// EvalTSErr evaluates TypeScript code and returns result or error.
func EvalTSErr(rt sdk.Runtime, source, code string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	pr, err := sdk.PublishKitEvalTS(rt, ctx, sdk.KitEvalTSMsg{Source: source, Code: code})
	if err != nil {
		return "", fmt.Errorf("publish kit.eval-ts: %w", err)
	}

	type evalResult struct {
		val string
		err error
	}
	ch := make(chan evalResult, 1)
	unsub, err := sdk.SubscribeKitEvalTSResp(rt, ctx, pr.ReplyTo,
		func(resp sdk.KitEvalTSResp, _ sdk.Message) {
			if resp.Error != "" {
				ch <- evalResult{err: fmt.Errorf("%s: %s", resp.Code, resp.Error)}
			} else {
				ch <- evalResult{val: resp.Result}
			}
		})
	if err != nil {
		return "", fmt.Errorf("subscribe kit.eval-ts reply: %w", err)
	}
	defer unsub()

	select {
	case r := <-ch:
		return r.val, r.err
	case <-ctx.Done():
		return "", fmt.Errorf("eval-ts %s: %w", source, ctx.Err())
	}
}

// SetDraining sets the draining state via kit.set-draining bus command.
func SetDraining(t *testing.T, rt sdk.Runtime, draining bool) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pr, err := sdk.PublishKitSetDraining(rt, ctx, sdk.KitSetDrainingMsg{Draining: draining})
	if err != nil {
		t.Fatalf("SetDraining: publish: %v", err)
	}

	ch := make(chan error, 1)
	unsub, err := sdk.SubscribeKitSetDrainingResp(rt, ctx, pr.ReplyTo,
		func(resp sdk.KitSetDrainingResp, _ sdk.Message) {
			if resp.Error != "" {
				ch <- fmt.Errorf("%s", resp.Error)
			} else {
				ch <- nil
			}
		})
	if err != nil {
		t.Fatalf("SetDraining: subscribe: %v", err)
	}
	defer unsub()

	select {
	case err := <-ch:
		if err != nil {
			t.Fatalf("SetDraining: %v", err)
		}
	case <-ctx.Done():
		t.Fatalf("SetDraining: timeout")
	}
}

// Teardown tears down a deployment via kit.teardown bus command.
func Teardown(t *testing.T, rt sdk.Runtime, source string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pr, err := sdk.PublishKitTeardown(rt, ctx, sdk.KitTeardownMsg{Source: source})
	if err != nil {
		t.Fatalf("Teardown(%s): publish: %v", source, err)
	}

	ch := make(chan error, 1)
	unsub, err := sdk.SubscribeKitTeardownResp(rt, ctx, pr.ReplyTo,
		func(resp sdk.KitTeardownResp, _ sdk.Message) {
			if resp.Error != "" {
				ch <- fmt.Errorf("%s", resp.Error)
			} else {
				ch <- nil
			}
		})
	if err != nil {
		t.Fatalf("Teardown(%s): subscribe: %v", source, err)
	}
	defer unsub()

	select {
	case err := <-ch:
		if err != nil {
			t.Fatalf("Teardown(%s): %v", source, err)
		}
	case <-ctx.Done():
		t.Fatalf("Teardown(%s): timeout", source)
	}
}

// ListDeployments lists current deployments via bus command.
func ListDeployments(t *testing.T, rt sdk.Runtime) []sdk.DeploymentInfo {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pr, err := sdk.PublishKitList(rt, ctx, sdk.KitListMsg{})
	if err != nil {
		t.Fatalf("ListDeployments: publish: %v", err)
	}

	ch := make(chan []sdk.DeploymentInfo, 1)
	unsub, err := sdk.SubscribeKitListResp(rt, ctx, pr.ReplyTo,
		func(resp sdk.KitListResp, _ sdk.Message) {
			ch <- resp.Deployments
		})
	if err != nil {
		t.Fatalf("ListDeployments: subscribe: %v", err)
	}
	defer unsub()

	select {
	case deps := <-ch:
		return deps
	case <-ctx.Done():
		t.Fatalf("ListDeployments: timeout")
		return nil
	}
}

// PublishAndWait publishes a typed message and waits for the raw reply payload.
func PublishAndWait(t *testing.T, rt sdk.Runtime, msg sdk.BrainkitMessage, timeout time.Duration) json.RawMessage {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	pr, err := sdk.Publish(rt, ctx, msg)
	if err != nil {
		t.Fatalf("PublishAndWait: publish %s: %v", msg.BusTopic(), err)
	}

	ch := make(chan json.RawMessage, 1)
	unsub, err := rt.SubscribeRaw(ctx, pr.ReplyTo, func(m sdk.Message) {
		select {
		case ch <- json.RawMessage(m.Payload):
		default:
		}
	})
	if err != nil {
		t.Fatalf("PublishAndWait: subscribe: %v", err)
	}
	defer unsub()

	select {
	case payload := <-ch:
		return payload
	case <-ctx.Done():
		t.Fatalf("PublishAndWait %s: timeout after %v", msg.BusTopic(), timeout)
		return nil
	}
}

// Schedule creates a schedule via the schedules.create bus command and returns the schedule ID.
func Schedule(t *testing.T, rt sdk.Runtime, expression, topic string, payload json.RawMessage) string {
	t.Helper()
	id, err := ScheduleErr(rt, expression, topic, payload)
	if err != nil {
		t.Fatalf("Schedule: %v", err)
	}
	return id
}

// ScheduleErr creates a schedule via bus command and returns the ID or error.
func ScheduleErr(rt sdk.Runtime, expression, topic string, payload json.RawMessage) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pr, err := sdk.PublishScheduleCreate(rt, ctx, sdk.ScheduleCreateMsg{
		Expression: expression,
		Topic:      topic,
		Payload:    payload,
	})
	if err != nil {
		return "", fmt.Errorf("publish schedules.create: %w", err)
	}

	type result struct {
		id  string
		err error
	}
	ch := make(chan result, 1)
	unsub, err := sdk.SubscribeScheduleCreateResp(rt, ctx, pr.ReplyTo,
		func(resp sdk.ScheduleCreateResp, _ sdk.Message) {
			if resp.Error != "" {
				ch <- result{err: fmt.Errorf("%s: %s", resp.Code, resp.Error)}
			} else {
				ch <- result{id: resp.ID}
			}
		})
	if err != nil {
		return "", fmt.Errorf("subscribe schedules.create reply: %w", err)
	}
	defer unsub()

	select {
	case r := <-ch:
		return r.id, r.err
	case <-ctx.Done():
		return "", fmt.Errorf("schedule: %w", ctx.Err())
	}
}

// Alive checks if the kit is healthy via the kit.health bus command.
func Alive(t *testing.T, rt sdk.Runtime) bool {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pr, err := sdk.PublishKitHealth(rt, ctx, sdk.KitHealthMsg{})
	if err != nil {
		return false
	}

	ch := make(chan bool, 1)
	unsub, err := sdk.SubscribeKitHealthResp(rt, ctx, pr.ReplyTo,
		func(resp sdk.KitHealthResp, _ sdk.Message) {
			ch <- resp.Error == ""
		})
	if err != nil {
		return false
	}
	defer unsub()

	select {
	case ok := <-ch:
		return ok
	case <-ctx.Done():
		return false
	}
}

// EvalModule deploys code as a module (for test framework support).
// Uses kit.deploy to evaluate code as a module, then tears it down if teardown is true.
func EvalModule(t *testing.T, rt sdk.Runtime, source, code string) {
	t.Helper()
	Deploy(t, rt, source, code)
}

// Unschedule cancels a schedule via the schedules.cancel bus command.
func Unschedule(t *testing.T, rt sdk.Runtime, id string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pr, err := sdk.PublishScheduleCancel(rt, ctx, sdk.ScheduleCancelMsg{ID: id})
	if err != nil {
		t.Fatalf("Unschedule(%s): publish: %v", id, err)
	}

	ch := make(chan error, 1)
	unsub, err := sdk.SubscribeScheduleCancelResp(rt, ctx, pr.ReplyTo,
		func(resp sdk.ScheduleCancelResp, _ sdk.Message) {
			if resp.Error != "" {
				ch <- fmt.Errorf("%s: %s", resp.Code, resp.Error)
			} else {
				ch <- nil
			}
		})
	if err != nil {
		t.Fatalf("Unschedule(%s): subscribe: %v", id, err)
	}
	defer unsub()

	select {
	case err := <-ch:
		if err != nil {
			t.Fatalf("Unschedule(%s): %v", id, err)
		}
	case <-ctx.Done():
		t.Fatalf("Unschedule(%s): timeout", id)
	}
}

// ListSchedules returns the list of active schedules via the schedules.list bus command.
func ListSchedules(t *testing.T, rt sdk.Runtime) []sdk.ScheduleInfo {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pr, err := sdk.PublishScheduleList(rt, ctx, sdk.ScheduleListMsg{})
	if err != nil {
		t.Fatalf("ListSchedules: publish: %v", err)
	}

	ch := make(chan []sdk.ScheduleInfo, 1)
	unsub, err := sdk.SubscribeScheduleListResp(rt, ctx, pr.ReplyTo,
		func(resp sdk.ScheduleListResp, _ sdk.Message) {
			ch <- resp.Schedules
		})
	if err != nil {
		t.Fatalf("ListSchedules: subscribe: %v", err)
	}
	defer unsub()

	select {
	case schedules := <-ch:
		return schedules
	case <-ctx.Done():
		t.Fatalf("ListSchedules: timeout")
		return nil
	}
}
