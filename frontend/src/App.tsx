import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
  type KeyboardEvent,
  type PointerEvent,
} from "react";
import { AnimatePresence, motion } from "motion/react";

import "./App.css";

type Point = {
  id: number;
  x: number;
  y: number;
  radius: number;
  phase: number;
};
type Position = Pick<Point, "x" | "y">;

const points: Point[] = [
  { id: 1, x: 16, y: 31, radius: 5, phase: 0.1 },
  { id: 2, x: 27, y: 19, radius: 3, phase: 1.4 },
  { id: 3, x: 38, y: 39, radius: 4, phase: 2.2 },
  { id: 4, x: 49, y: 25, radius: 6, phase: 4.1 },
  { id: 5, x: 62, y: 33, radius: 3, phase: 0.7 },
  { id: 6, x: 76, y: 17, radius: 4, phase: 3.2 },
  { id: 7, x: 86, y: 36, radius: 5, phase: 5.3 },
  { id: 8, x: 72, y: 52, radius: 3, phase: 2.7 },
  { id: 9, x: 89, y: 67, radius: 3, phase: 1.9 },
  { id: 10, x: 59, y: 69, radius: 5, phase: 4.8 },
  { id: 11, x: 43, y: 58, radius: 3, phase: 3.6 },
  { id: 12, x: 29, y: 74, radius: 4, phase: 5.9 },
  { id: 13, x: 11, y: 61, radius: 3, phase: 2.9 },
];

const initialPositions = Object.fromEntries(
  points.map(({ id, x, y }) => [id, { x, y }]),
) as Record<number, Position>;

const deepStars = Array.from({ length: 72 }, (_, index) => ({
  id: index,
  x: (index * 83 + 47) % 1000,
  y: (index * 137 + 91) % 720,
  radius: index % 11 === 0 ? 1.8 : index % 4 === 0 ? 1.1 : 0.65,
}));

/** Returns whether the visitor has requested less motion. */
function useReducedMotion() {
  const [reduced, setReduced] = useState(
    () => window.matchMedia("(prefers-reduced-motion: reduce)").matches,
  );

  useEffect(() => {
    const query = window.matchMedia("(prefers-reduced-motion: reduce)");
    const update = () => setReduced(query.matches);
    query.addEventListener("change", update);
    return () => query.removeEventListener("change", update);
  }, []);

  return reduced;
}

/** Renders Aether's public coming-soon constellation. */
function App() {
  const reducedMotion = useReducedMotion();
  const [positions, setPositions] = useState(initialPositions);
  const [selected, setSelected] = useState<number | null>(null);
  const [dragging, setDragging] = useState<number | null>(null);
  const [arcEpoch, setArcEpoch] = useState(0);
  const fieldRef = useRef<SVGSVGElement>(null);
  const pointerRef = useRef<Position | null>(null);

  const selectPoint = useCallback((id: number) => {
    setSelected(id);
    setArcEpoch((epoch) => epoch + 1);
  }, []);

  useEffect(() => {
    if (reducedMotion) return;
    let frame = 0;
    let previous = 0;
    const animate = (time: number) => {
      if (time - previous > 34) {
        previous = time;
        setPositions((current) => {
          const next = { ...current };
          for (const point of points) {
            if (point.id === dragging) continue;
            const home = initialPositions[point.id];
            const position = current[point.id];
            const pointer = pointerRef.current;
            let gravityX = 0;
            let gravityY = 0;
            if (pointer) {
              const dx = pointer.x - position.x;
              const dy = pointer.y - position.y;
              const distance = Math.hypot(dx, dy);
              if (distance < 24 && distance > 0) {
                const pull = (1 - distance / 24) * 0.012;
                gravityX = dx * pull;
                gravityY = dy * pull;
              }
            }
            next[point.id] = {
              x:
                position.x +
                (home.x - position.x) * 0.012 +
                Math.cos(time / 6200 + point.phase) * 0.018 +
                gravityX,
              y:
                position.y +
                (home.y - position.y) * 0.012 +
                Math.sin(time / 7600 + point.phase) * 0.014 +
                gravityY,
            };
          }
          return next;
        });
      }
      frame = requestAnimationFrame(animate);
    };
    frame = requestAnimationFrame(animate);
    return () => cancelAnimationFrame(frame);
  }, [dragging, reducedMotion]);

  const toFieldPosition = useCallback((clientX: number, clientY: number) => {
    const bounds = fieldRef.current?.getBoundingClientRect();
    if (!bounds) return null;
    return {
      x: ((clientX - bounds.left) / bounds.width) * 100,
      y: ((clientY - bounds.top) / bounds.height) * 100,
    };
  }, []);

  const handlePointerMove = (event: PointerEvent<SVGSVGElement>) => {
    const pointer = toFieldPosition(event.clientX, event.clientY);
    pointerRef.current = pointer;
    if (dragging !== null && pointer) {
      setPositions((current) => ({ ...current, [dragging]: pointer }));
    }
  };

  const selectedPoint = selected === null ? null : positions[selected];
  const neighbours = useMemo(() => {
    if (selected === null || !selectedPoint) return [];
    return points
      .filter((point) => point.id !== selected)
      .map((point) => ({
        id: point.id,
        distance: Math.hypot(
          positions[point.id].x - selectedPoint.x,
          positions[point.id].y - selectedPoint.y,
        ),
      }))
      .sort((a, b) => a.distance - b.distance)
      .slice(0, 3);
  }, [positions, selected, selectedPoint]);

  const handleKeyDown = (event: KeyboardEvent<SVGGElement>, id: number) => {
    if (event.key === "Enter" || event.key === " ") {
      event.preventDefault();
      selectPoint(id);
    }
  };

  return (
    <main className="archive">
      <div className="registration registration--north" aria-hidden="true" />
      <div className="registration registration--east" aria-hidden="true" />
      <header className="masthead">
        <h1>AETHER</h1>
        <p>COMING SOON</p>
      </header>

      <svg
        ref={fieldRef}
        className="constellation"
        viewBox="0 0 1000 720"
        preserveAspectRatio="none"
        role="group"
        aria-label="Interactive document constellation"
        data-motion={reducedMotion ? "reduced" : "full"}
        onPointerMove={handlePointerMove}
        onPointerLeave={() => {
          pointerRef.current = null;
          setDragging(null);
        }}
        onPointerUp={() => setDragging(null)}
      >
        <defs>
          <filter id="soft-light" x="-80%" y="-80%" width="260%" height="260%">
            <feGaussianBlur stdDeviation="8" />
          </filter>
          <filter id="signal-light" x="-100%" y="-100%" width="300%" height="300%">
            <feGaussianBlur stdDeviation="3.5" result="blur" />
            <feMerge>
              <feMergeNode in="blur" />
              <feMergeNode in="SourceGraphic" />
            </feMerge>
          </filter>
        </defs>

        <g className="deep-field" aria-hidden="true">
          {deepStars.map((star) => (
            <circle
              key={star.id}
              cx={star.x}
              cy={star.y}
              r={star.radius}
              className={star.id % 7 === 0 ? "deep-star deep-star--signal" : "deep-star"}
            />
          ))}
        </g>

        <g className="galaxy" aria-hidden="true">
          <ellipse className="galaxy__haze" cx="620" cy="420" rx="280" ry="74" />
          <g className="galaxy__disc galaxy__disc--outer">
            <ellipse cx="620" cy="420" rx="305" ry="102" />
            <ellipse cx="620" cy="420" rx="260" ry="82" />
          </g>
          <g className="galaxy__disc galaxy__disc--inner">
            <ellipse cx="620" cy="420" rx="205" ry="55" />
            <ellipse cx="620" cy="420" rx="148" ry="34" />
          </g>
          <ellipse className="galaxy__event" cx="620" cy="420" rx="78" ry="18" />
          <circle className="galaxy__axis" cx="620" cy="420" r="6" />
        </g>

        <g className="atlas-rules" aria-hidden="true">
          <path d="M 74 80 H 260 M 74 80 V 208" />
          <path d="M 926 640 H 748 M 926 640 V 516" />
          <circle cx="500" cy="360" r="208" />
          <circle cx="500" cy="360" r="308" />
          <path d="M 500 42 V 678 M 128 360 H 872" />
          <path d="M 490 78 H 510 M 490 642 H 510 M 172 350 V 370 M 828 350 V 370" />
        </g>
        <g className="base-links" aria-hidden="true">
          <path d="M 160 223 Q 223 106 270 137" />
          <path d="M 270 137 Q 332 274 380 281" />
          <path d="M 380 281 Q 440 142 490 180" />
          <path d="M 490 180 Q 556 264 620 238" />
          <path d="M 620 238 Q 696 92 760 122" />
          <path d="M 760 122 Q 832 192 860 259" />
          <path d="M 860 259 Q 818 348 720 374" />
          <path d="M 720 374 Q 780 514 890 482" />
          <path d="M 720 374 Q 655 474 590 497" />
          <path d="M 590 497 Q 510 397 430 418" />
          <path d="M 430 418 Q 365 548 290 533" />
          <path d="M 290 533 Q 174 498 110 439" />
        </g>

        <AnimatePresence>
          {selectedPoint && (
            <motion.g
              key={`lens-${selected}`}
              className="gravity-lens"
              aria-hidden="true"
              initial={{ opacity: 0, scale: 0.35 }}
              animate={{ opacity: 1, scale: 1 }}
              exit={{ opacity: 0, scale: 1.8 }}
              transition={{ type: "spring", stiffness: 120, damping: 18 }}
              style={{ transformOrigin: `${selectedPoint.x * 10}px ${selectedPoint.y * 7.2}px` }}
            >
              <circle cx={selectedPoint.x * 10} cy={selectedPoint.y * 7.2} r="34" />
              <circle cx={selectedPoint.x * 10} cy={selectedPoint.y * 7.2} r="52" />
            </motion.g>
          )}
        </AnimatePresence>

        <AnimatePresence>
          {selectedPoint && neighbours.map(({ id }, index) => {
            const target = positions[id];
            const x1 = selectedPoint.x * 10;
            const y1 = selectedPoint.y * 7.2;
            const x2 = target.x * 10;
            const y2 = target.y * 7.2;
            return (
              <motion.path
                key={`${selected}-${id}-${arcEpoch}`}
                className="selection-arc"
                d={`M ${x1} ${y1} Q ${(x1 + x2) / 2} ${(y1 + y2) / 2 + (index % 2 === 0 ? -46 : 46)} ${x2} ${y2}`}
                aria-hidden="true"
                initial={{ opacity: 0, pathLength: 0 }}
                animate={{ opacity: [0, 1, 0.72, 0], pathLength: 1 }}
                exit={{ opacity: 0 }}
                transition={{ duration: 3.1, times: [0, 0.12, 0.62, 1] }}
              />
            );
          })}
        </AnimatePresence>

        {points.map((point) => {
          const position = positions[point.id];
          const active = selected === point.id;
          return (
            <g
              key={point.id}
              className={`star${active ? " star--active" : ""}`}
              role="button"
              aria-label={`Constellation point ${point.id}`}
              aria-pressed={active}
              tabIndex={0}
              transform={`translate(${position.x * 10} ${position.y * 7.2})`}
              onKeyDown={(event) => handleKeyDown(event, point.id)}
              onPointerDown={(event) => {
                event.currentTarget.setPointerCapture(event.pointerId);
                setDragging(point.id);
                selectPoint(point.id);
              }}
            >
              <circle className="star__target" r="24" />
              <circle className="star__orbit" r={point.radius + 9} />
              <circle className="star__core" r={point.radius} />
              <path className="star__tick" d="M -18 0 H -10 M 10 0 H 18" />
            </g>
          );
        })}
      </svg>
      <div className="signal" aria-hidden="true">
        <span />
      </div>
    </main>
  );
}

export default App;
