import { useEffect, useMemo, useRef, useState } from "react";
import { Terminal } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit";
import { WebLinksAddon } from "@xterm/addon-web-links";
import "@xterm/xterm/css/xterm.css";
import { applyMobileKey, MOBILE_KEYS } from "./mobileKeys";

type Pod = { name: string; container: string[]; ready: boolean; node: string; age: string };
type Info = { target: { kind: string; name: string; namespace: string } };

export function App() {
  const termContainer = useRef<HTMLDivElement | null>(null);
  const wsRef = useRef<WebSocket | null>(null);
  const termRef = useRef<Terminal | null>(null);
  const fitRef = useRef<FitAddon | null>(null);
  const [pods, setPods] = useState<Pod[]>([]);
  const [pod, setPod] = useState("");
  const [container, setContainer] = useState("");
  const [info, setInfo] = useState<Info | null>(null);
  const [status, setStatus] = useState("idle");
  const [modifier, setModifier] = useState({ ctrl: false, alt: false });

  useEffect(() => {
    Promise.all([fetch("/api/v1/pods").then((r) => r.json()), fetch("/api/v1/info").then((r) => r.json())]).then(
      ([p, i]) => {
        setPods(p);
        setInfo(i);
        if (p.length > 0) {
          setPod(p[0].name);
          setContainer(p[0].container[0] ?? "");
        }
      }
    );
  }, []);

  useEffect(() => {
    const term = new Terminal({ convertEol: true, cursorBlink: true });
    const fit = new FitAddon();
    term.loadAddon(fit);
    term.loadAddon(new WebLinksAddon());
    termRef.current = term;
    fitRef.current = fit;
    if (termContainer.current) {
      term.open(termContainer.current);
      fit.fit();
    }
    const onResize = () => {
      fit.fit();
      sendControl({ type: "resize", cols: term.cols, rows: term.rows });
    };
    window.addEventListener("resize", onResize);
    window.addEventListener("orientationchange", onResize);
    return () => {
      window.removeEventListener("resize", onResize);
      window.removeEventListener("orientationchange", onResize);
      term.dispose();
    };
  }, []);

  const selectedPod = useMemo(() => pods.find((p) => p.name === pod), [pods, pod]);

  function connect() {
    if (!pod) return;
    const query = new URLSearchParams({ pod, container }).toString();
    const ws = new WebSocket(`${location.origin.replace(/^http/, "ws")}/api/v1/exec?${query}`);
    ws.binaryType = "arraybuffer";
    wsRef.current = ws;
    setStatus("connecting");
    ws.onopen = () => {
      setStatus("connected");
      const term = termRef.current;
      if (!term) return;
      term.focus();
      sendControl({ type: "resize", cols: term.cols, rows: term.rows });
      term.onData((d) => {
        if (ws.readyState !== WebSocket.OPEN) return;
        const payload = new Uint8Array(1 + d.length);
        payload[0] = 0x00;
        for (let i = 0; i < d.length; i += 1) payload[i + 1] = d.charCodeAt(i);
        ws.send(payload);
      });
    };
    ws.onmessage = (event) => {
      const data = new Uint8Array(event.data as ArrayBuffer);
      if (data[0] === 0x00) {
        termRef.current?.write(new TextDecoder().decode(data.slice(1)));
      } else if (data[0] === 0x02) {
        const msg = JSON.parse(new TextDecoder().decode(data.slice(1)));
        if (msg.type === "exit") setStatus("session ended");
        if (msg.type === "error") setStatus(`error: ${msg.msg}`);
      }
    };
    ws.onclose = () => setStatus("session ended");
  }

  function sendControl(payload: object) {
    if (!wsRef.current || wsRef.current.readyState !== WebSocket.OPEN) return;
    const body = new TextEncoder().encode(JSON.stringify(payload));
    const frame = new Uint8Array(body.length + 1);
    frame[0] = 0x01;
    frame.set(body, 1);
    wsRef.current.send(frame);
  }

  return (
    <div className="mx-auto flex min-h-screen max-w-6xl flex-col gap-4 p-4">
      <h1 className="text-2xl font-semibold">kubeshell-web</h1>
      <div className="text-sm opacity-80">
        Target: {info?.target.kind}/{info?.target.name} in {info?.target.namespace}
      </div>
      <div className="flex gap-2">
        <select
          className="rounded bg-slate-800 p-2"
          value={pod}
          onChange={(e) => {
            setPod(e.target.value);
            const p = pods.find((item) => item.name === e.target.value);
            setContainer(p?.container[0] ?? "");
          }}
        >
          {pods.map((p) => (
            <option key={p.name} value={p.name}>
              {p.name}
            </option>
          ))}
        </select>
        <select className="rounded bg-slate-800 p-2" value={container} onChange={(e) => setContainer(e.target.value)}>
          {(selectedPod?.container ?? []).map((c) => (
            <option key={c}>{c}</option>
          ))}
        </select>
        <button className="rounded bg-blue-600 px-3 py-2" onClick={connect}>
          {status === "session ended" ? "Reconnect" : "Connect"}
        </button>
      </div>
      <div className="rounded border border-slate-700 bg-black p-2">
        <div ref={termContainer} className="h-[52vh] min-w-0 overflow-hidden" />
      </div>
      <div className="overflow-x-auto whitespace-nowrap rounded border border-slate-700 p-2">
        {MOBILE_KEYS.map((key) => (
          <button
            key={key}
            className="mr-2 rounded bg-slate-700 px-3 py-2 text-sm"
            onClick={() => {
              const result = applyMobileKey(key, modifier);
              setModifier(result.state);
              if (result.output) {
                termRef.current?.input(result.output);
              }
            }}
          >
            {key}
          </button>
        ))}
      </div>
      <div className="text-sm">{status}</div>
    </div>
  );
}
