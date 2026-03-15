// Theme management
(function () {
  "use strict";

  const STORAGE_KEY = "fpvladder-theme";
  const DEFAULT_THEME = "light";

  // Calculate relative path from current page to index.html
  // Uses the location of scripts.js as anchor point
  function getRelativePathToRoot() {
    // Find the path to scripts.js - it's in the same dir as index.html
    const scripts = document.querySelectorAll('script[src*="scripts.js"]');
    for (const script of scripts) {
      const src = script.getAttribute("src");
      if (src) {
        // src is relative path from current page to scripts.js
        // index.html is in the same dir as scripts.js
        const lastSlash = src.lastIndexOf("/");
        if (lastSlash >= 0) {
          return src.substring(0, lastSlash + 1);
        } else {
          // scripts.js is in the same dir as current page
          return "./";
        }
      }
    }
    // Fallback: calculate from pathname
    const path = window.location.pathname;
    const parts = path.split("/").filter((p) => p.length > 0);
    const lastPart = parts[parts.length - 1];
    if (lastPart && lastPart.includes(".")) {
      parts.pop();
    }
    // Remove first part if it's a drive letter or protocol (file:)
    if (parts.length > 0 && parts[0].includes(":")) {
      parts.shift();
    }
    const depth = parts.length;
    return depth === 0 ? "./" : "../".repeat(depth);
  }

  // Get saved theme or default
  function getSavedTheme() {
    try {
      return localStorage.getItem(STORAGE_KEY) || DEFAULT_THEME;
    } catch (e) {
      return DEFAULT_THEME;
    }
  }

  // Save theme to localStorage
  function saveTheme(theme) {
    try {
      localStorage.setItem(STORAGE_KEY, theme);
    } catch (e) {
      // Ignore storage errors
    }
  }

  // Apply theme to document
  function applyTheme(theme) {
    document.documentElement.setAttribute("data-theme", theme);
    updateThemeButtons(theme);
  }

  // Update active state of theme options
  function updateThemeButtons(theme) {
    document.querySelectorAll(".theme-option").forEach((btn) => {
      btn.classList.toggle("active", btn.dataset.value === theme);
    });
  }

  // Toggle settings dropdown
  function toggleDropdown() {
    const dropdown = document.getElementById("settingsDropdown");
    if (dropdown) {
      dropdown.classList.toggle("active");
    }
  }

  // Close dropdown when clicking outside
  function closeDropdownOnClickOutside(e) {
    const settingsBtn = document.getElementById("settingsBtn");
    const settingsClose = document.getElementById("settingsClose");
    const dropdown = document.getElementById("settingsDropdown");

    if (
      dropdown &&
      !dropdown.contains(e.target) &&
      !settingsBtn.contains(e.target) &&
      !(settingsClose && settingsClose.contains(e.target))
    ) {
      dropdown.classList.remove("active");
    }
  }

  // Initialize theme
  function init() {
    const savedTheme = getSavedTheme();
    applyTheme(savedTheme);

    // Setup event listeners
    document.addEventListener("DOMContentLoaded", function () {
      // Setup back arrow link and visibility
      const backArrow = document.getElementById("backArrow");
      const rootPath = getRelativePathToRoot() + "index.html";

      if (backArrow) {
        const path = window.location.pathname;
        const isHomePage =
          document.body.dataset.hasBack === "false" ||
          path === "/" ||
          path.endsWith("/index.html");
        if (isHomePage) {
          backArrow.style.display = "none";
        } else {
          backArrow.style.display = "flex";
          backArrow.href = rootPath;
        }
      }

      // Settings button click
      const settingsBtn = document.getElementById("settingsBtn");
      if (settingsBtn) {
        settingsBtn.addEventListener("click", toggleDropdown);
      }

      // Close button click
      const settingsClose = document.getElementById("settingsClose");
      if (settingsClose) {
        settingsClose.addEventListener("click", (e) => {
          e.stopPropagation();
          const dropdown = document.getElementById("settingsDropdown");
          if (dropdown) dropdown.classList.remove("active");
        });
      }

      // Close dropdown on outside click
      document.addEventListener("click", closeDropdownOnClickOutside);

      // Theme option clicks
      document.querySelectorAll(".theme-option").forEach((btn) => {
        btn.addEventListener("click", function () {
          const theme = this.dataset.value;
          if (theme) {
            applyTheme(theme);
            saveTheme(theme);
            // Close dropdown after theme change
            const dropdown = document.getElementById("settingsDropdown");
            if (dropdown) dropdown.classList.remove("active");
          }
        });
      });
    });
  }

  // Run initialization
  init();
})();

// Row click handler for data-href
document.addEventListener("click", (e) => {
  const target = e.target.closest("[data-href]");
  if (target && !e.target.closest("a")) {
    window.location.href = target.dataset.href;
  }
});

// Smart header - hide on scroll down, show on scroll up
(function () {
  let lastScrollY = window.scrollY;
  let ticking = false;
  const header = document.querySelector(".site-header");
  const headerHeight = 50;

  function updateHeader() {
    const currentScrollY = window.scrollY;

    if (currentScrollY <= 0) {
      // At top - always show
      header.style.transform = "translateY(0)";
    } else if (currentScrollY > lastScrollY && currentScrollY > headerHeight) {
      // Scrolling down - hide
      header.style.transform = "translateY(-100%)";
    } else if (currentScrollY < lastScrollY) {
      // Scrolling up - show
      header.style.transform = "translateY(0)";
    }

    lastScrollY = currentScrollY;
    ticking = false;
  }

  window.addEventListener(
    "scroll",
    function () {
      if (!ticking) {
        window.requestAnimationFrame(updateHeader);
        ticking = true;
      }
    },
    { passive: true },
  );
})();
