(function(){
  "use strict";
  var nav=document.getElementById("nav");
  var burger=document.getElementById("navBurger");
  var menu=document.getElementById("navMenu");
  if(!nav||!burger||!menu)return;
  function setMenu(open){
    nav.classList.toggle("menu-open",open);
    menu.classList.toggle("open",open);
    burger.setAttribute("aria-expanded",String(open));
    burger.setAttribute("aria-label",open?"Close menu":"Open menu");
  }
  burger.addEventListener("click",function(event){event.stopPropagation();setMenu(!menu.classList.contains("open"));});
  menu.addEventListener("click",function(event){if(event.target.closest("a"))setMenu(false);});
  document.addEventListener("click",function(event){if(menu.classList.contains("open")&&!menu.contains(event.target)&&!burger.contains(event.target))setMenu(false);});
  document.addEventListener("keydown",function(event){if(event.key==="Escape")setMenu(false);});
  window.matchMedia("(min-width:901px)").addEventListener("change",function(event){if(event.matches)setMenu(false);});

  var theme=document.getElementById("themeBtn");
  if(theme){theme.addEventListener("click",function(){try{localStorage.setItem("dockly-site-theme",document.documentElement.dataset.theme)}catch(error){}});}
})();
