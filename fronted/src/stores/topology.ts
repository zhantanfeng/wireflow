import { defineStore } from 'pinia'
import { getTopology } from '@/api/monitor'

export interface TopologyNode {
    id: string | number;
    name: string;
    ip: string;
    x: number;
    y: number;
    status: 'online' | 'offline';
    type: string;
}

export interface TopologyLink {
    id: string | number;
    from: string | number;
    to: string | number;
    quality: 'good' | 'warn' | 'error';
    latency: number;
}

export const useTopologyStore = defineStore('topology', {
    state: () => ({
        nodes: [] as TopologyNode[],
        links: [] as TopologyLink[],
        loading: false,
        viewState: {
            scale: 1,
            offset: { x: 0, y: 0 }
        },
        selectedNodeId: null as string | number | null,
        nodeMetrics: {
            cpu_trend: [22, 25, 21, 28, 32, 35, 33, 30, 28, 26, 25, 28, 30, 32, 35, 38, 40, 38, 35, 33],
            mem_trend: [45, 45, 46, 46, 46, 47, 47, 48, 48, 48, 48, 49, 49, 49, 50, 50, 51, 51, 51, 52],
            net_tx: [45, 52, 48, 60, 85, 92, 78, 65, 55, 62, 70, 88, 95, 82, 75, 68, 60, 55, 50, 58],
            net_rx: [12, 15, 14, 18, 25, 30, 28, 22, 19, 21, 24, 28, 32, 29, 26, 23, 20, 18, 16, 19],
            timestamps: [
                '14:00:02', '14:00:04', '14:00:06', '14:00:08', '14:00:10',
                '14:00:12', '14:00:14', '14:00:16', '14:00:18', '14:00:20',
                '14:00:22', '14:00:24', '14:00:26', '14:00:28', '14:00:30',
                '14:00:32', '14:00:34', '14:00:36', '14:00:38', '14:00:40'
            ]
        }
    }),

    actions: {
        async fetchTopology() {
            this.loading = true
            try {
                const res: any = await getTopology()
                const topology = res?.data

                if (topology?.nodes?.length) {
                    this.nodes = topology.nodes
                    this.links = topology.links || []
                } else {
                    this.generateMockTopology()
                }
            } catch (error) {
                console.error('Failed to load topology, falling back to mock data.', error)
                this.generateMockTopology()
            } finally {
                this.loading = false
            }
        },

        selectNode(id: string | number) {
            this.selectedNodeId = id
        },

        tickNodeMetrics() {
            const now = new Date()
            const timeStr = now.toLocaleTimeString([], { hour12: false, minute: '2-digit', second: '2-digit' })

            this.nodeMetrics.timestamps.shift()
            this.nodeMetrics.cpu_trend.shift()
            this.nodeMetrics.net_tx.shift()
            this.nodeMetrics.net_rx.shift()
            this.nodeMetrics.mem_trend.shift()

            this.nodeMetrics.timestamps.push(timeStr)
            this.nodeMetrics.cpu_trend.push(Math.floor(Math.random() * 15 + 25))
            this.nodeMetrics.net_tx.push(Math.floor(Math.random() * 40 + 40))
            this.nodeMetrics.net_rx.push(Math.floor(Math.random() * 20 + 10))
            this.nodeMetrics.mem_trend.push(50 + Math.floor(Math.random() * 2))
        },

        generateMockTopology() {
            this.nodes = [
                { id: 'n1', name: 'Center-Gateway', ip: '10.24.0.1', x: 420, y: 260, status: 'online', type: 'relay' },
                { id: 'n2', name: 'Edge-Shanghai', ip: '10.24.0.5', x: 150, y: 120, status: 'online', type: 'edge' },
                { id: 'n3', name: 'Edge-Beijing', ip: '10.24.0.8', x: 690, y: 120, status: 'online', type: 'edge' },
                { id: 'n4', name: 'HK-User-iMac', ip: '10.24.5.12', x: 420, y: 450, status: 'offline', type: 'client' },
            ]

            this.links = [
                { id: 'l1', from: 'n1', to: 'n2', quality: 'good', latency: 24 },
                { id: 'l2', from: 'n1', to: 'n3', quality: 'good', latency: 35 },
                { id: 'l3', from: 'n1', to: 'n4', quality: 'error', latency: 0 },
            ]
        },

        updateZoom(delta: number) {
            const newScale = Math.min(2.2, Math.max(0.6, +(this.viewState.scale + delta).toFixed(2)))
            this.viewState.scale = newScale
        },

        resetView() {
            this.viewState.scale = 1
            this.viewState.offset = { x: 0, y: 0 }
        }
    }
})
