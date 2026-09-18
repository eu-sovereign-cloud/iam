document.querySelectorAll("[data-copy]").forEach(function (button) {
  button.addEventListener("click", function () {
    var target = document.getElementById(button.getAttribute("data-copy"));
    if (!target) return;
    navigator.clipboard.writeText(target.textContent.trim()).then(function () {
      var original = button.textContent;
      button.textContent = "Copied";
      button.classList.add("link-button-confirmed");
      setTimeout(function () {
        button.textContent = original;
        button.classList.remove("link-button-confirmed");
      }, 1600);
    });
  });
});
