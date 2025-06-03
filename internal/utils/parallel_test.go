package utils

import (
	"context"
	"testing"
)

// 为了测试上下文取消而定义的任务类型
type TestTaskWithContext struct {
	ID  int
	Ctx context.Context
}

// 定义测试专用的结果类型
type TestTaskProcessResult struct {
	ID     int
	Status string
	Value  int
}

func TestParallelMap(t *testing.T) {
	//// 测试空输入
	//t.Run("empty input", func(t *testing.T) {
	//	var emptyInput []int
	//	result := ParallelMap(emptyInput, 4, func(i int) int {
	//		return i * 2
	//	})
	//	if len(result) != 0 {
	//		t.Errorf("expected empty result, got %v", result)
	//	}
	//})
	//
	//// 测试单元素输入 - 应该直接处理，不使用并发
	//t.Run("single input", func(t *testing.T) {
	//	input := []int{42}
	//	result := ParallelMap(input, 4, func(i int) int {
	//		return i * 2
	//	})
	//	if len(result) != 1 || result[0] != 84 {
	//		t.Errorf("expected [84], got %v", result)
	//	}
	//})
	//
	//// 测试多元素输入 - 确保顺序正确
	//t.Run("multiple inputs with order", func(t *testing.T) {
	//	input := []int{1, 2, 3, 4, 5}
	//	expected := []int{2, 4, 6, 8, 10}
	//
	//	result := ParallelMap(input, 3, func(i int) int {
	//		// 添加随机延迟，测试顺序保持
	//		time.Sleep(time.Duration(rand.Intn(10)) * time.Millisecond)
	//		return i * 2
	//	})
	//
	//	if !reflect.DeepEqual(result, expected) {
	//		t.Errorf("expected %v, got %v", expected, result)
	//	}
	//})
	//
	//// 测试并发执行 - 确保真的是并行处理
	//t.Run("concurrent execution", func(t *testing.T) {
	//	input := make([]int, 100)
	//	for i := range input {
	//		input[i] = i
	//	}
	//
	//	var maxConcurrent int32
	//	var currentConcurrent int32
	//
	//	ParallelMap(input, 10, func(i int) int {
	//		// 原子操作增加当前并发计数
	//		current := atomic.AddInt32(&currentConcurrent, 1)
	//		// 更新最大并发数
	//		for {
	//			max := atomic.LoadInt32(&maxConcurrent)
	//			if current <= max {
	//				break
	//			}
	//			if atomic.CompareAndSwapInt32(&maxConcurrent, max, current) {
	//				break
	//			}
	//		}
	//
	//		// 模拟工作
	//		time.Sleep(20 * time.Millisecond)
	//
	//		// 完成工作，减少计数
	//		atomic.AddInt32(&currentConcurrent, -1)
	//		return i * 2
	//	})
	//
	//	// 检查最大并发数是否接近预期
	//	if maxConcurrent < 5 || maxConcurrent > 10 {
	//		t.Errorf("expected max concurrent between 5-10, got %d", maxConcurrent)
	//	}
	//})
	//
	//// 测试大量任务的性能
	//t.Run("performance with many tasks", func(t *testing.T) {
	//	const taskCount = 10000
	//	input := make([]int, taskCount)
	//	for i := range input {
	//		input[i] = i
	//	}
	//
	//	start := time.Now()
	//	result := ParallelMap(input, 16, func(i int) int {
	//		// 非常轻量的工作
	//		return i * i
	//	})
	//	duration := time.Since(start)
	//
	//	// 验证结果正确性
	//	for i, v := range result {
	//		if v != i*i {
	//			t.Errorf("incorrect result at index %d: expected %d, got %d", i, i*i, v)
	//			break
	//		}
	//	}
	//
	//	t.Logf("Processed %d tasks in %v", taskCount, duration)
	//})
	//
	//// 测试带有上下文的任务
	//t.Run("tasks with context cancellation", func(t *testing.T) {
	//	// 创建一个父上下文
	//	parentCtx, parentCancel := context.WithCancel(context.Background())
	//	defer parentCancel()
	//
	//	// 创建一个包含50个任务的切片
	//	const taskCount = 50
	//	tasks := make([]TestTaskWithContext, taskCount)
	//
	//	// 一半任务使用可取消的上下文，一半使用父上下文
	//	for i := 0; i < taskCount; i++ {
	//		if i%2 == 0 {
	//			// 偶数任务使用子上下文，将被取消
	//			childCtx, _ := context.WithCancel(parentCtx)
	//			tasks[i] = TestTaskWithContext{ID: i, Ctx: childCtx}
	//		} else {
	//			// 奇数任务使用父上下文，不会被取消
	//			tasks[i] = TestTaskWithContext{ID: i, Ctx: parentCtx}
	//		}
	//	}
	//
	//	// 在单独协程中稍后取消上下文
	//	go func() {
	//		time.Sleep(50 * time.Millisecond)
	//		t.Log("Cancelling parent context")
	//		parentCancel()
	//	}()
	//
	//	// 使用一个计数器跟踪有多少任务被取消了
	//	var canceledCount int32
	//	var completedCount int32
	//
	//	// 启动并行处理
	//	results := ParallelMap(tasks, 8, func(task TestTaskWithContext) TestTaskProcessResult {
	//		// 模拟工作负载 - 第一阶段
	//		time.Sleep(time.Duration(30+rand.Intn(40)) * time.Millisecond)
	//
	//		// 检查上下文是否已取消
	//		if err := task.Ctx.Err(); err != nil {
	//			atomic.AddInt32(&canceledCount, 1)
	//			return TestTaskProcessResult{
	//				ID:     task.ID,
	//				Status: "canceled",
	//				Value:  -1,
	//			}
	//		}
	//
	//		// 模拟工作负载 - 第二阶段
	//		time.Sleep(time.Duration(30+rand.Intn(40)) * time.Millisecond)
	//
	//		// 再次检查上下文，以防在处理过程中被取消
	//		if err := task.Ctx.Err(); err != nil {
	//			atomic.AddInt32(&canceledCount, 1)
	//			return TestTaskProcessResult{
	//				ID:     task.ID,
	//				Status: "canceled_during_processing",
	//				Value:  -1,
	//			}
	//		}
	//
	//		// 正常完成任务
	//		atomic.AddInt32(&completedCount, 1)
	//		return TestTaskProcessResult{
	//			ID:     task.ID,
	//			Status: "completed",
	//			Value:  task.ID * 10, // 一些计算结果
	//		}
	//	})
	//
	//	// 确认所有任务都被处理
	//	if len(results) != taskCount {
	//		t.Errorf("Expected %d results, got %d", taskCount, len(results))
	//	}
	//
	//	// 检查取消和完成的数量
	//	t.Logf("Tasks completed: %d, canceled: %d", completedCount, canceledCount)
	//
	//	// 由于上下文被取消，应该有一些任务被取消
	//	if canceledCount == 0 {
	//		t.Errorf("Expected some tasks to be canceled, but none were")
	//	}
	//
	//	// 验证结果：每个任务都应该有对应的结果，要么完成要么取消
	//	for i, result := range results {
	//		if result.ID != i {
	//			t.Errorf("Result ID mismatch at index %d: expected %d, got %d", i, i, result.ID)
	//		}
	//
	//		// 检查状态是否符合预期
	//		if result.Status == "completed" {
	//			if result.Value != i*10 {
	//				t.Errorf("Unexpected result value for completed task %d: %d", i, result.Value)
	//			}
	//		} else if !contains([]string{"canceled", "canceled_during_processing"}, result.Status) {
	//			t.Errorf("Unexpected status for task %d: %s", i, result.Status)
	//		}
	//	}
	//})
}

// 辅助函数：检查切片是否包含某个字符串
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
