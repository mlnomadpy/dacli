/* Native anchors remain readable without JavaScript; enhancement adds presentation controls. */
(() => {
  const slides = [...document.querySelectorAll(".slide")];
  const select = document.querySelector("#slide-select");
  const previous = document.querySelector("#previous");
  const next = document.querySelector("#next");
  let current = 0;
  slides.forEach((slide, index) => {
    const option = document.createElement("option");
    option.value = String(index);
    option.textContent = `${index + 1}. ${slide.querySelector(".slide-label").textContent}`;
    select.append(option);
  });
  function show() {
    const index = slides.findIndex((slide) => `#${slide.id}` === location.hash);
    current = index < 0 ? 0 : index;
    slides.forEach((slide, i) => {
      slide.hidden = i !== current;
    });
    select.value = String(current);
    previous.disabled = current === 0;
    next.disabled = current === slides.length - 1;
    document.querySelector("#count").textContent =
      `${current + 1} / ${slides.length}`;
    window.scrollTo(0, 0);
    document.title = `dacli — ${slides[current].querySelector(".slide-label").textContent}`;
  }
  function go(index) {
    const bounded = Math.max(0, Math.min(slides.length - 1, index));
    location.hash = slides[bounded].id;
  }
  previous.addEventListener("click", () => go(current - 1));
  next.addEventListener("click", () => go(current + 1));
  select.addEventListener("change", () => go(Number(select.value)));
  document.addEventListener("keydown", (event) => {
    if (
      event.altKey ||
      event.ctrlKey ||
      event.metaKey ||
      event.target.closest(
        "input, select, textarea, button, a, [contenteditable]",
      )
    )
      return;
    const destinations = {
      ArrowRight: current + 1,
      ArrowLeft: current - 1,
      PageDown: current + 1,
      PageUp: current - 1,
      Home: 0,
      End: slides.length - 1,
    };
    if (event.key in destinations) {
      event.preventDefault();
      go(destinations[event.key]);
    }
  });
  window.addEventListener("hashchange", show);
  document.querySelector("#print").hidden = false;
  document
    .querySelector("#print")
    .addEventListener("click", () => window.print());
  document.querySelector(".controls").hidden = false;
  document.body.classList.add("enhanced");
  show();
})();
