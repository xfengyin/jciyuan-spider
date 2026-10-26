package main

import "testing"

// 这个测试是**故意失败的**，唯一目的是验证分支保护的必需检查真的能挡住合并。
// 它只存在于临时分支 test/ci-gate-proof 上；取证完成后该分支即删除，不进 main。
//
// 判据：本测试失败 ⇒ job test 失败 ⇒ job ci-summary（needs: [test], if: always()）失败
//       ⇒ 必需检查 ci-summary 在 PR head 上 conclusion=failure
//       ⇒ 该 PR 的 mergeable_state 应为 blocked。
func TestDeliberateFailureForGateProof(t *testing.T) {
	t.Fatal("这段失败是故意的：用于证明必需检查 ci-summary 失败时 PR 合不进去")
}
