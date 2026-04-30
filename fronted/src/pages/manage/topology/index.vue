<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { storeToRefs } from 'pinia'
import { ZoomIn, ZoomOut, Maximize2, Activity, Server, TriangleAlert, Waypoints } from 'lucide-vue-next'
import { useTopologyStore } from '@/stores/topology'

definePage({
  meta: { title: 'Workspace Topology', description: 'Visualize workspace peer connectivity.' },
})

const topologyStore = useTopologyStore()
const { nodes, links, loading, viewState, nodeMetrics, selectedNodeId } = storeToRefs(topologyStore)
const selectedId = ref<string | number | null>(null)
let metricTimer: ReturnType<typeof setInterval> | null = null

const viewBox = computed(() => {
  const width = 960 / viewState.value.scale
  const height = 560 / viewState.value.scale
  return `${viewState.value.offset.x} ${viewState.value.offset.y} ${width} ${height}`
})

const stats = computed(() => {
  const online = nodes.value.filter(node => node.status === 'online').length
  const offline = nodes.value.filter(node => node.status === 'offline').length
  const goodLinks = links.value.filter(link => link.quality === 'good').length

  return {
    total: nodes.value.length,
    online,
    offline,
    links: links.value.length,
    goodLinks,
  }
})

const selectedNode = computed(() => {
  const id = selectedId.value ?? selectedNodeId.value
  return nodes.value.find(node => node.id === id) || null
})

const activePeers = computed(() => {
  if (!selectedNode.value) return []

  return links.value
    .filter(link => link.from === selectedNode.value?.id || link.to === selectedNode.value?.id)
    .map(link => {
      const peerId = link.from === selectedNode.value?.id ? link.to : link.from
      const peerNode = nodes.value.find(node => node.id === peerId)
      return {
        ...link,
        peerName: peerNode?.name || String(peerId),
      }
    })
})

function linkPath(link: { from: string | number; to: string | number }) {
  const source = nodes.value.find(node => node.id === link.from)
  const target = nodes.value.find(node => node.id === link.to)
  if (!source || !target) return ''

  const dx = target.x - source.x
  const dy = target.y - source.y
  const cx = (source.x + target.x) / 2 - dy * 0.12
  const cy = (source.y + target.y) / 2 + dx * 0.12
  return `M ${source.x} ${source.y} Q ${cx} ${cy} ${target.x} ${target.y}`
}

function nodeColor(type: string, status: string) {
  if (status === 'offline') return '#71717a'
  if (type === 'relay') return '#22d3ee'
  if (type === 'client') return '#a78bfa'
  return '#34d399'
}

function qualityColor(quality: string) {
  if (quality === 'good') return '#22d3ee'
  if (quality === 'warn') return '#f59e0b'
  return '#f87171'
}

function selectNode(id: string | number) {
  selectedId.value = id
  topologyStore.selectNode(id)
}

function zoomIn() {
  topologyStore.updateZoom(0.1)
}

function zoomOut() {
  topologyStore.updateZoom(-0.1)
}

function resetView() {
  topologyStore.resetView()
}

onMounted(() => {
  topologyStore.fetchTopology()
  metricTimer = setInterval(() => topologyStore.tickNodeMetrics(), 2000)
})

onUnmounted(() => {
  if (metricTimer) {
    clearInterval(metricTimer)
  }
})
</script>

<template>
  <div class="flex h-full flex-col bg-zinc-950 text-zinc-100">
    <div class="border-b border-zinc-800 px-6 py-4">
      <div class="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
        <div class="flex items-center gap-3">
          <div class="rounded-xl border border-cyan-500/20 bg-cyan-500/10 p-2 text-cyan-300">
            <Waypoints class="size-5" />
          </div>
          <div>
            <h1 class="text-2xl font-semibold tracking-tight">Workspace Topology</h1>
            <p class="text-sm text-zinc-400">Derived from peers and controller computed peer config.</p>
          </div>
        </div>

        <div class="flex items-center gap-2">
          <button class="flex size-9 items-center justify-center rounded-lg border border-zinc-700 bg-zinc-900 text-zinc-300 hover:border-zinc-500" @click="zoomOut">
            <ZoomOut class="size-4" />
          </button>
          <button class="flex size-9 items-center justify-center rounded-lg border border-zinc-700 bg-zinc-900 text-zinc-300 hover:border-zinc-500" @click="zoomIn">
            <ZoomIn class="size-4" />
          </button>
          <button class="flex size-9 items-center justify-center rounded-lg border border-zinc-700 bg-zinc-900 text-zinc-300 hover:border-zinc-500" @click="resetView">
            <Maximize2 class="size-4" />
          </button>
        </div>
      </div>

      <div class="mt-4 grid gap-3 md:grid-cols-4">
        <div class="rounded-xl border border-zinc-800 bg-zinc-900/60 p-4">
          <div class="text-xs uppercase tracking-[0.18em] text-zinc-500">Nodes</div>
          <div class="mt-2 text-2xl font-semibold">{{ stats.total }}</div>
          <div class="mt-1 text-xs text-zinc-400">{{ stats.online }} online / {{ stats.offline }} offline</div>
        </div>
        <div class="rounded-xl border border-zinc-800 bg-zinc-900/60 p-4">
          <div class="text-xs uppercase tracking-[0.18em] text-zinc-500">Links</div>
          <div class="mt-2 text-2xl font-semibold">{{ stats.links }}</div>
          <div class="mt-1 text-xs text-zinc-400">{{ stats.goodLinks }} healthy links</div>
        </div>
        <div class="rounded-xl border border-zinc-800 bg-zinc-900/60 p-4">
          <div class="text-xs uppercase tracking-[0.18em] text-zinc-500">Traffic</div>
          <div class="mt-2 flex items-center gap-2 text-2xl font-semibold text-cyan-300">
            <Activity class="size-5" />
            {{ nodeMetrics.net_tx.at(-1) }} Mb/s
          </div>
          <div class="mt-1 text-xs text-zinc-400">Synthetic live metric preview</div>
        </div>
        <div class="rounded-xl border border-zinc-800 bg-zinc-900/60 p-4">
          <div class="text-xs uppercase tracking-[0.18em] text-zinc-500">Status</div>
          <div class="mt-2 flex items-center gap-2 text-2xl font-semibold">
            <TriangleAlert class="size-5 text-amber-300" />
            {{ loading ? 'Syncing' : 'Ready' }}
          </div>
          <div class="mt-1 text-xs text-zinc-400">Falls back to mock data when the API is unavailable.</div>
        </div>
      </div>
    </div>

    <div class="grid min-h-0 flex-1 gap-0 lg:grid-cols-[minmax(0,1fr)_320px]">
      <div class="relative min-h-0 overflow-hidden border-r border-zinc-800 bg-[#09090b]">
        <div class="absolute inset-0 bg-[radial-gradient(circle_at_1px_1px,rgba(255,255,255,0.08)_1px,transparent_0)] bg-[size:28px_28px] opacity-40" />

        <svg class="relative h-full w-full" :viewBox="viewBox">
          <defs>
            <filter id="nodeGlow" x="-50%" y="-50%" width="200%" height="200%">
              <feGaussianBlur stdDeviation="4" result="blur" />
              <feMerge>
                <feMergeNode in="blur" />
                <feMergeNode in="SourceGraphic" />
              </feMerge>
            </filter>
          </defs>

          <g v-for="link in links" :key="link.id">
            <path
              :d="linkPath(link)"
              fill="none"
              :stroke="qualityColor(link.quality)"
              :stroke-width="selectedNode && (link.from === selectedNode.id || link.to === selectedNode.id) ? 2.6 : 1.6"
              :stroke-dasharray="link.quality === 'error' ? '6 5' : undefined"
              :stroke-opacity="link.quality === 'error' ? 0.45 : 0.75"
            />
          </g>

          <g v-for="node in nodes" :key="node.id" class="cursor-pointer" @click="selectNode(node.id)">
            <circle
              :cx="node.x"
              :cy="node.y"
              :r="node.type === 'relay' ? 22 : 18"
              :fill="`${nodeColor(node.type, node.status)}20`"
              :stroke="nodeColor(node.type, node.status)"
              stroke-width="2"
              filter="url(#nodeGlow)"
            />
            <circle
              :cx="node.x"
              :cy="node.y"
              :r="node.type === 'relay' ? 30 : 25"
              fill="transparent"
              :stroke="selectedNode?.id === node.id ? '#ffffff55' : '#ffffff18'"
              stroke-width="1.4"
            />
            <text
              :x="node.x"
              :y="node.y + 40"
              text-anchor="middle"
              class="fill-zinc-100 text-[12px] font-semibold"
            >
              {{ node.name }}
            </text>
            <text
              :x="node.x"
              :y="node.y + 55"
              text-anchor="middle"
              class="fill-zinc-500 text-[10px] font-mono"
            >
              {{ node.ip || 'pending-ip' }}
            </text>
          </g>
        </svg>

        <div v-if="loading && !nodes.length" class="absolute inset-0 flex items-center justify-center bg-zinc-950/70">
          <div class="rounded-xl border border-zinc-800 bg-zinc-900/90 px-4 py-3 text-sm text-zinc-300">Loading topology...</div>
        </div>
      </div>

      <div class="flex min-h-0 flex-col bg-zinc-950">
        <div class="border-b border-zinc-800 px-5 py-4">
          <div class="text-xs uppercase tracking-[0.18em] text-zinc-500">Node Inspector</div>
          <div class="mt-2 flex items-center gap-2">
            <Server class="size-4 text-zinc-500" />
            <span class="text-sm font-medium text-zinc-200">{{ selectedNode?.name || 'Select a node' }}</span>
          </div>
        </div>

        <div v-if="selectedNode" class="min-h-0 flex-1 overflow-y-auto px-5 py-4">
          <div class="space-y-4">
            <div class="rounded-xl border border-zinc-800 bg-zinc-900/60 p-4">
              <div class="flex items-center justify-between">
                <div>
                  <div class="text-lg font-semibold text-zinc-100">{{ selectedNode.name }}</div>
                  <div class="mt-1 font-mono text-xs text-zinc-500">{{ selectedNode.ip || 'pending-ip' }}</div>
                </div>
                <span
                  class="rounded-full px-2.5 py-1 text-[11px] font-medium uppercase tracking-wide"
                  :class="selectedNode.status === 'online'
                    ? 'bg-emerald-500/15 text-emerald-300'
                    : 'bg-zinc-700 text-zinc-300'"
                >
                  {{ selectedNode.status }}
                </span>
              </div>

              <div class="mt-4 grid grid-cols-2 gap-3 text-sm">
                <div class="rounded-lg bg-zinc-950/80 p-3">
                  <div class="text-[11px] uppercase tracking-wide text-zinc-500">Type</div>
                  <div class="mt-1 font-medium text-zinc-200">{{ selectedNode.type }}</div>
                </div>
                <div class="rounded-lg bg-zinc-950/80 p-3">
                  <div class="text-[11px] uppercase tracking-wide text-zinc-500">Peers</div>
                  <div class="mt-1 font-medium text-zinc-200">{{ activePeers.length }}</div>
                </div>
                <div class="rounded-lg bg-zinc-950/80 p-3">
                  <div class="text-[11px] uppercase tracking-wide text-zinc-500">CPU</div>
                  <div class="mt-1 font-medium text-cyan-300">{{ nodeMetrics.cpu_trend.at(-1) }}%</div>
                </div>
                <div class="rounded-lg bg-zinc-950/80 p-3">
                  <div class="text-[11px] uppercase tracking-wide text-zinc-500">TX</div>
                  <div class="mt-1 font-medium text-cyan-300">{{ nodeMetrics.net_tx.at(-1) }} Mb/s</div>
                </div>
              </div>
            </div>

            <div class="rounded-xl border border-zinc-800 bg-zinc-900/60 p-4">
              <div class="text-sm font-medium text-zinc-200">Active Links</div>
              <div class="mt-3 space-y-2">
                <div
                  v-for="link in activePeers"
                  :key="link.id"
                  class="flex items-center justify-between rounded-lg border border-zinc-800 bg-zinc-950/70 px-3 py-2"
                >
                  <div>
                    <div class="text-sm text-zinc-100">{{ link.peerName }}</div>
                    <div class="mt-0.5 text-xs text-zinc-500">{{ link.latency }} ms</div>
                  </div>
                  <span
                    class="rounded-full px-2 py-1 text-[11px] font-medium uppercase"
                    :class="link.quality === 'good'
                      ? 'bg-cyan-500/15 text-cyan-300'
                      : link.quality === 'warn'
                        ? 'bg-amber-500/15 text-amber-300'
                        : 'bg-red-500/15 text-red-300'"
                  >
                    {{ link.quality }}
                  </span>
                </div>
                <div v-if="!activePeers.length" class="rounded-lg border border-dashed border-zinc-800 px-3 py-4 text-sm text-zinc-500">
                  No computed peer links for this node.
                </div>
              </div>
            </div>
          </div>
        </div>

        <div v-else class="flex flex-1 items-center justify-center px-6 text-center text-sm text-zinc-500">
          Click any node in the topology graph to inspect its derived peer links.
        </div>
      </div>
    </div>
  </div>
</template>
