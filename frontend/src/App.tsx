import { Suspense, lazy } from "react";

import VaultApp from "./VaultApp";

const ApiBench = lazy(() => import("./ApiBench"));

/** Routes the private vault and keeps the raw API bench on its deliberate developer path. */
function App() {
  if (window.location.pathname.replace(/\/$/, "") === "/dev/api-bench") {
    return (
      <Suspense fallback={null}>
        <ApiBench />
      </Suspense>
    );
  }

  return <VaultApp />;
}

export default App;
