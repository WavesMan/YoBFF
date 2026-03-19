import { RequestError } from '../../admin/api'
import type { WeaverDAG, WeaverVersion } from '../../admin/types'

/**
 *
 * buildEmptyDag 创建空DAG结构，用于初始化与重置编排状态。
 *
 */
export function buildEmptyDag(): WeaverDAG {
  return {
    nodes: [],
    edges: [],
    output_node_id: '',
  }
}

/**
 *
 * buildVersionSnapshot 构建版本详情快照，用于详情面板展示。
 *
 */
export function buildVersionSnapshot(version: WeaverVersion) {
  return {
    inputs: version.inputs || [],
    mapping: version.mapping || {},
    dag: version.dag || buildEmptyDag(),
    node_contracts: version.node_contracts || [],
  }
}

/**
 *
 * resolveWeaverRequestError 解析后端错误码，返回可直接展示的中文提示。
 *
 */
export function resolveWeaverRequestError(error: unknown, fallback: string) {
  if (error instanceof RequestError) {
    if (error.error_code === 'version_not_found') {
      return '版本不存在，请刷新版本列表后重试'
    }
    if (error.error_code === 'weaver_run_failed') {
      return '运行失败，请检查输入数据、映射规则与DAG配置'
    }
    if (error.error_code === 'weaver_invalid_dag') {
      return 'DAG结构无效，请检查节点与连线关系'
    }
    return error.message || fallback
  }
  if (error instanceof Error) {
    return error.message
  }
  return fallback
}
