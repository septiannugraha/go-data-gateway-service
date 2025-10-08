// LKPP Branding Customization Script
// This script adds INAPROC branding and removes Fusio branding from the account app

(function brandCaption(){
  function enhanceOnce(img){
    if (!img || img.dataset.lkppEnhanced) return;
    img.dataset.lkppEnhanced = '1';
    // Center the parent container so the logo + caption are centered
    var p = img.parentElement;
    if (p) {
      p.style.display = 'flex';
      p.style.flexDirection = 'column';
      p.style.alignItems = 'center';
      p.style.justifyContent = 'center';
    }
    var wrap = document.createElement('div');
    wrap.className = 'lkpp-brandstack';
    wrap.innerHTML = '<div class="lkpp-brandline"><span class="ina">INA</span><span class="proc">PROC</span></div>'+
                     '<div class="lkpp-sub">API Gateway</div>';
    img.insertAdjacentElement('afterend', wrap);
  }

  function removeAccountButtons(){
    document.querySelectorAll('a, button').forEach(function(el){
      var text = (el.textContent || '').trim();
      if (!text) return;
      if (/^account$/i.test(text) && (el.classList.contains('btn') || el.classList.contains('nav-link') || el.closest('.navbar'))){
        el.remove();
      }
    });
  }

  function removeFusioBranding(){
    // Remove elements that are purely Fusio branded labels
    document.querySelectorAll('a, span, small, div, footer').forEach(function(el){
      var t = (el.textContent || '').trim();
      if (/^fusio$/i.test(t)) {
        el.remove();
      }
    });
    // Remove isolated Fusio text nodes
    var walker = document.createTreeWalker(document.body, NodeFilter.SHOW_TEXT, null);
    var nodes = [];
    while (walker.nextNode()) nodes.push(walker.currentNode);
    nodes.forEach(function(node){
      if (/fusio/i.test(node.nodeValue)){
        var cleaned = node.nodeValue.replace(/fusio/gi, '').replace(/\s{2,}/g, ' ').trim();
        if (!cleaned) {
          // If completely empty, hide parent container if it looks like a brand/badge
          var pe = node.parentElement;
          if (pe && (/(brand|navbar|badge|footer)/i.test(pe.className || '') || pe.tagName === 'SMALL')) {
            pe.remove();
            return;
          }
        }
        node.nodeValue = cleaned;
      }
    });
    // Remove attributes mentioning Fusio
    document.querySelectorAll('[title], [aria-label], [alt]').forEach(function(el){
      ['title','aria-label','alt'].forEach(function(attr){
        var v = el.getAttribute(attr);
        if (v && /fusio/i.test(v)) el.setAttribute(attr, v.replace(/fusio/gi, '').trim());
      });
    });
  }

  function alignPasswordReset(){
    var candidates = Array.from(document.querySelectorAll('a, button'));
    candidates.forEach(function(el){
      var t = (el.textContent || '').trim().toLowerCase();
      if (t === 'password reset' || t === 'reset password' || t === 'forgot password' || t === 'lupa kata sandi' || t === 'lupa password'){
        el.classList.add('lkpp-reset-link');
        var row = el.parentElement;
        if (row) {
          row.style.display = 'flex';
          row.style.alignItems = 'center';
          row.style.gap = '10px';
        }
      }
    });
  }

  function scan(){
    // Target all logo variants: lkpp-logo, fusio_64px, fusio_32px
    document.querySelectorAll('img[src*="lkpp-logo"], img[src*="fusio_64px"], img[src*="fusio_32px"]').forEach(enhanceOnce);
    removeAccountButtons();
    removeFusioBranding();
    alignPasswordReset();
  }

  document.addEventListener('DOMContentLoaded', scan);
  new MutationObserver(function(){ scan(); }).observe(document.documentElement, {childList:true, subtree:true});
})();
