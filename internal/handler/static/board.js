(function () {
  const canvas = document.getElementById("canvas");
  const ctx = canvas.getContext("2d");
  const colorInput = document.getElementById("color");
  const widthInput = document.getElementById("width");
  const clearButton = document.getElementById("clear");

  let drawing = false;
  let lastX = 0;
  let lastY = 0;

  const wsScheme = window.location.protocol === "https:" ? "wss" : "ws";
  const ws = new WebSocket(`${wsScheme}://${window.location.host}/ws/${window.SESSION_ID}`);

  function resizeCanvas() {
    const rect = canvas.getBoundingClientRect();
    canvas.width = rect.width;
    canvas.height = rect.height;
  }

  function drawStroke(stroke) {
    ctx.strokeStyle = stroke.color || "#111";
    ctx.lineWidth = stroke.width || 3;
    ctx.lineCap = "round";
    ctx.lineJoin = "round";
    ctx.beginPath();
    ctx.moveTo(stroke.x0, stroke.y0);
    ctx.lineTo(stroke.x1, stroke.y1);
    ctx.stroke();
  }

  function sendStroke(x0, y0, x1, y1) {
    if (ws.readyState !== WebSocket.OPEN) {
      return;
    }
    ws.send(JSON.stringify({
      type: "stroke",
      color: colorInput.value,
      width: Number(widthInput.value),
      x0,
      y0,
      x1,
      y1,
    }));
  }

  ws.onmessage = (event) => {
    const msg = JSON.parse(event.data);
    if (msg.type === "history") {
      resizeCanvas();
      (msg.strokes || []).forEach(drawStroke);
    } else if (msg.type === "stroke") {
      drawStroke(msg);
    } else if (msg.type === "clear") {
      ctx.clearRect(0, 0, canvas.width, canvas.height);
    }
  };

  function pointerPos(event) {
    const rect = canvas.getBoundingClientRect();
    return {
      x: event.clientX - rect.left,
      y: event.clientY - rect.top,
    };
  }

  canvas.addEventListener("pointerdown", (event) => {
    drawing = true;
    canvas.setPointerCapture(event.pointerId);
    const pos = pointerPos(event);
    lastX = pos.x;
    lastY = pos.y;
  });

  canvas.addEventListener("pointermove", (event) => {
    if (!drawing) {
      return;
    }
    const pos = pointerPos(event);
    drawStroke({
      color: colorInput.value,
      width: Number(widthInput.value),
      x0: lastX,
      y0: lastY,
      x1: pos.x,
      y1: pos.y,
    });
    sendStroke(lastX, lastY, pos.x, pos.y);
    lastX = pos.x;
    lastY = pos.y;
  });

  function stopDrawing(event) {
    if (!drawing) {
      return;
    }
    drawing = false;
    canvas.releasePointerCapture(event.pointerId);
  }

  canvas.addEventListener("pointerup", stopDrawing);
  canvas.addEventListener("pointerleave", stopDrawing);

  clearButton.addEventListener("click", () => {
    if (ws.readyState === WebSocket.OPEN) {
      ws.send(JSON.stringify({ type: "clear" }));
    }
    ctx.clearRect(0, 0, canvas.width, canvas.height);
  });

  window.addEventListener("resize", resizeCanvas);
  resizeCanvas();
})();
