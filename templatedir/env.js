document.addEventListener("DOMContentLoaded", function () {
  const els = document.querySelectorAll(`script[type="application/json"][data-env]`);
  const s = (els[0] && els[0].textContent) || "{}";
  Object.defineProperty(window, "ENV", {
    value:
      els.length === 1
        ? function (name) {
            try {
              const data = JSON.parse(s);
              return name === undefined ? data : data[name];
            } catch {
              return undefined;
            }
          }
        : function () {
            return Error("refusing to get value, more than one element with attribute data-env defined");
          },
  });
});
