// Force-directed layout for the Graph tab. The simulation runs to completion
// up front (no animation), so the view renders once per layout instead of on
// every tick.
import {
  forceCollide,
  forceLink,
  forceManyBody,
  forceSimulation,
  forceX,
  forceY,
  type SimulationLinkDatum,
  type SimulationNodeDatum,
} from "d3-force";
import type { GraphEdge, GraphNode } from "./graph";

export interface Point {
  x: number;
  y: number;
}

interface SimNode extends SimulationNodeDatum {
  id: string;
}

/**
 * Lays out nodes and edges. Nodes found in previous keep their position as a
 * starting point and the simulation starts cooler, so changing a filter
 * nudges the picture instead of reshuffling it.
 */
export function layoutGraph(nodes: GraphNode[], edges: GraphEdge[], previous?: Map<string, Point>): Map<string, Point> {
  let known = 0;
  const simNodes: SimNode[] = nodes.map((n) => {
    const p = previous?.get(n.id);
    if (p) known++;
    return p ? { id: n.id, x: p.x, y: p.y } : { id: n.id };
  });
  const links: SimulationLinkDatum<SimNode>[] = edges.map((e) => ({ source: e.source, target: e.target }));
  const n = simNodes.length;

  const sim = forceSimulation(simNodes)
    .force(
      "link",
      forceLink<SimNode, SimulationLinkDatum<SimNode>>(links)
        .id((d) => d.id)
        .distance(36)
        .strength(0.5),
    )
    .force("charge", forceManyBody<SimNode>().strength(n > 400 ? -45 : -80).distanceMax(500))
    // Weak pull to the center keeps disconnected nodes from drifting off.
    .force("x", forceX<SimNode>(0).strength(0.06))
    .force("y", forceY<SimNode>(0).strength(0.06))
    .force("collide", forceCollide<SimNode>(9))
    .stop();

  const warm = n > 0 && known / n > 0.7;
  if (warm) sim.alpha(0.3);
  const ticks = warm ? 120 : n > 500 ? 220 : 300;
  for (let i = 0; i < ticks; i++) sim.tick();

  const out = new Map<string, Point>();
  for (const s of simNodes) out.set(s.id, { x: s.x ?? 0, y: s.y ?? 0 });
  return out;
}

/** Bounding box of points, padded; a unit box around the origin when empty. */
export function bounds(points: Iterable<Point>, pad = 24) {
  let minX = Infinity,
    minY = Infinity,
    maxX = -Infinity,
    maxY = -Infinity;
  for (const p of points) {
    minX = Math.min(minX, p.x);
    minY = Math.min(minY, p.y);
    maxX = Math.max(maxX, p.x);
    maxY = Math.max(maxY, p.y);
  }
  if (minX === Infinity) return { x: -50, y: -50, w: 100, h: 100 };
  return { x: minX - pad, y: minY - pad, w: maxX - minX + 2 * pad, h: maxY - minY + 2 * pad };
}
