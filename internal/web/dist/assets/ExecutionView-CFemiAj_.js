import{d as A,o as D,z as H,h as N,c as h,a as f,t as x,n as m,e as T,b as y,F as G,j as V,A as M,r as p,q as S,B as z,u as W,g as _,C as X}from"./index-OyecHHAX.js";var v=function(r,e){return Object.defineProperty?Object.defineProperty(r,"raw",{value:e}):r.raw=e,r},l;(function(r){r[r.EOS=0]="EOS",r[r.Text=1]="Text",r[r.Incomplete=2]="Incomplete",r[r.ESC=3]="ESC",r[r.Unknown=4]="Unknown",r[r.SGR=5]="SGR",r[r.OSCURL=6]="OSCURL"})(l||(l={}));class J{constructor(){this.VERSION="6.0.6",this.setup_palettes(),this._use_classes=!1,this.bold=!1,this.faint=!1,this.italic=!1,this.underline=!1,this.fg=this.bg=null,this._buffer="",this._url_allowlist={http:1,https:1},this._escape_html=!0,this.boldStyle="font-weight:bold",this.faintStyle="opacity:0.7",this.italicStyle="font-style:italic",this.underlineStyle="text-decoration:underline"}set use_classes(e){this._use_classes=e}get use_classes(){return this._use_classes}set url_allowlist(e){this._url_allowlist=e}get url_allowlist(){return this._url_allowlist}set escape_html(e){this._escape_html=e}get escape_html(){return this._escape_html}set boldStyle(e){this._boldStyle=e}get boldStyle(){return this._boldStyle}set faintStyle(e){this._faintStyle=e}get faintStyle(){return this._faintStyle}set italicStyle(e){this._italicStyle=e}get italicStyle(){return this._italicStyle}set underlineStyle(e){this._underlineStyle=e}get underlineStyle(){return this._underlineStyle}setup_palettes(){this.ansi_colors=[[{rgb:[0,0,0],class_name:"ansi-black"},{rgb:[187,0,0],class_name:"ansi-red"},{rgb:[0,187,0],class_name:"ansi-green"},{rgb:[187,187,0],class_name:"ansi-yellow"},{rgb:[0,0,187],class_name:"ansi-blue"},{rgb:[187,0,187],class_name:"ansi-magenta"},{rgb:[0,187,187],class_name:"ansi-cyan"},{rgb:[255,255,255],class_name:"ansi-white"}],[{rgb:[85,85,85],class_name:"ansi-bright-black"},{rgb:[255,85,85],class_name:"ansi-bright-red"},{rgb:[0,255,0],class_name:"ansi-bright-green"},{rgb:[255,255,85],class_name:"ansi-bright-yellow"},{rgb:[85,85,255],class_name:"ansi-bright-blue"},{rgb:[255,85,255],class_name:"ansi-bright-magenta"},{rgb:[85,255,255],class_name:"ansi-bright-cyan"},{rgb:[255,255,255],class_name:"ansi-bright-white"}]],this.palette_256=[],this.ansi_colors.forEach(n=>{n.forEach(t=>{this.palette_256.push(t)})});let e=[0,95,135,175,215,255];for(let n=0;n<6;++n)for(let t=0;t<6;++t)for(let a=0;a<6;++a){let i={rgb:[e[n],e[t],e[a]],class_name:"truecolor"};this.palette_256.push(i)}let s=8;for(let n=0;n<24;++n,s+=10){let t={rgb:[s,s,s],class_name:"truecolor"};this.palette_256.push(t)}}escape_txt_for_html(e){return this._escape_html?e.replace(/[&<>"']/gm,s=>{if(s==="&")return"&amp;";if(s==="<")return"&lt;";if(s===">")return"&gt;";if(s==='"')return"&quot;";if(s==="'")return"&#x27;"}):e}append_buffer(e){var s=this._buffer+e;this._buffer=s}get_next_packet(){var e={kind:l.EOS,text:"",url:""},s=this._buffer.length;if(s==0)return e;var n=this._buffer.indexOf("\x1B");if(n==-1)return e.kind=l.Text,e.text=this._buffer,this._buffer="",e;if(n>0)return e.kind=l.Text,e.text=this._buffer.slice(0,n),this._buffer=this._buffer.slice(n),e;if(n==0){if(s<3)return e.kind=l.Incomplete,e;var t=this._buffer.charAt(1);if(t!="["&&t!="]"&&t!="(")return e.kind=l.ESC,e.text=this._buffer.slice(0,1),this._buffer=this._buffer.slice(1),e;if(t=="["){this._csi_regex||(this._csi_regex=O(B||(B=v([`
                        ^                           # beginning of line
                                                    #
                                                    # First attempt
                        (?:                         # legal sequence
                          \x1B[                      # CSI
                          ([<-?]?)              # private-mode char
                          ([d;]*)                    # any digits or semicolons
                          ([ -/]?               # an intermediate modifier
                          [@-~])                # the command
                        )
                        |                           # alternate (second attempt)
                        (?:                         # illegal sequence
                          \x1B[                      # CSI
                          [ -~]*                # anything legal
                          ([\0-:])              # anything illegal
                        )
                    `],[`
                        ^                           # beginning of line
                                                    #
                                                    # First attempt
                        (?:                         # legal sequence
                          \\x1b\\[                      # CSI
                          ([\\x3c-\\x3f]?)              # private-mode char
                          ([\\d;]*)                    # any digits or semicolons
                          ([\\x20-\\x2f]?               # an intermediate modifier
                          [\\x40-\\x7e])                # the command
                        )
                        |                           # alternate (second attempt)
                        (?:                         # illegal sequence
                          \\x1b\\[                      # CSI
                          [\\x20-\\x7e]*                # anything legal
                          ([\\x00-\\x1f:])              # anything illegal
                        )
                    `]))));let i=this._buffer.match(this._csi_regex);if(i===null)return e.kind=l.Incomplete,e;if(i[4])return e.kind=l.ESC,e.text=this._buffer.slice(0,1),this._buffer=this._buffer.slice(1),e;i[1]!=""||i[3]!="m"?e.kind=l.Unknown:e.kind=l.SGR,e.text=i[2];var a=i[0].length;return this._buffer=this._buffer.slice(a),e}else if(t=="]"){if(s<4)return e.kind=l.Incomplete,e;if(this._buffer.charAt(2)!="8"||this._buffer.charAt(3)!=";")return e.kind=l.ESC,e.text=this._buffer.slice(0,1),this._buffer=this._buffer.slice(1),e;this._osc_st||(this._osc_st=Q(I||(I=v([`
                        (?:                         # legal sequence
                          (\x1B\\)                    # ESC                           |                           # alternate
                          (\x07)                      # BEL (what xterm did)
                        )
                        |                           # alternate (second attempt)
                        (                           # illegal sequence
                          [\0-]                 # anything illegal
                          |                           # alternate
                          [\b-]                 # anything illegal
                          |                           # alternate
                          [-]                 # anything illegal
                        )
                    `],[`
                        (?:                         # legal sequence
                          (\\x1b\\\\)                    # ESC \\
                          |                           # alternate
                          (\\x07)                      # BEL (what xterm did)
                        )
                        |                           # alternate (second attempt)
                        (                           # illegal sequence
                          [\\x00-\\x06]                 # anything illegal
                          |                           # alternate
                          [\\x08-\\x1a]                 # anything illegal
                          |                           # alternate
                          [\\x1c-\\x1f]                 # anything illegal
                        )
                    `])))),this._osc_st.lastIndex=0;{let u=this._osc_st.exec(this._buffer);if(u===null)return e.kind=l.Incomplete,e;if(u[3])return e.kind=l.ESC,e.text=this._buffer.slice(0,1),this._buffer=this._buffer.slice(1),e}{let u=this._osc_st.exec(this._buffer);if(u===null)return e.kind=l.Incomplete,e;if(u[3])return e.kind=l.ESC,e.text=this._buffer.slice(0,1),this._buffer=this._buffer.slice(1),e}this._osc_regex||(this._osc_regex=O(j||(j=v([`
                        ^                           # beginning of line
                                                    #
                        \x1B]8;                    # OSC Hyperlink
                        [ -:<-~]*       # params (excluding ;)
                        ;                           # end of params
                        ([!-~]{0,512})        # URL capture
                        (?:                         # ST
                          (?:\x1B\\)                  # ESC                           |                           # alternate
                          (?:\x07)                    # BEL (what xterm did)
                        )
                        ([ -~]+)              # TEXT capture
                        \x1B]8;;                   # OSC Hyperlink End
                        (?:                         # ST
                          (?:\x1B\\)                  # ESC                           |                           # alternate
                          (?:\x07)                    # BEL (what xterm did)
                        )
                    `],[`
                        ^                           # beginning of line
                                                    #
                        \\x1b\\]8;                    # OSC Hyperlink
                        [\\x20-\\x3a\\x3c-\\x7e]*       # params (excluding ;)
                        ;                           # end of params
                        ([\\x21-\\x7e]{0,512})        # URL capture
                        (?:                         # ST
                          (?:\\x1b\\\\)                  # ESC \\
                          |                           # alternate
                          (?:\\x07)                    # BEL (what xterm did)
                        )
                        ([\\x20-\\x7e]+)              # TEXT capture
                        \\x1b\\]8;;                   # OSC Hyperlink End
                        (?:                         # ST
                          (?:\\x1b\\\\)                  # ESC \\
                          |                           # alternate
                          (?:\\x07)                    # BEL (what xterm did)
                        )
                    `]))));let i=this._buffer.match(this._osc_regex);if(i===null)return e.kind=l.ESC,e.text=this._buffer.slice(0,1),this._buffer=this._buffer.slice(1),e;e.kind=l.OSCURL,e.url=i[1],e.text=i[2];var a=i[0].length;return this._buffer=this._buffer.slice(a),e}else if(t=="(")return e.kind=l.Unknown,this._buffer=this._buffer.slice(3),e}}ansi_to_html(e){this.append_buffer(e);for(var s=[];;){var n=this.get_next_packet();if(n.kind==l.EOS||n.kind==l.Incomplete)break;n.kind==l.ESC||n.kind==l.Unknown||(n.kind==l.Text?s.push(this.transform_to_html(this.with_state(n))):n.kind==l.SGR?this.process_ansi(n):n.kind==l.OSCURL&&s.push(this.process_hyperlink(n)))}return s.join("")}with_state(e){return{bold:this.bold,faint:this.faint,italic:this.italic,underline:this.underline,fg:this.fg,bg:this.bg,text:e.text}}process_ansi(e){let s=e.text.split(";");for(;s.length>0;){let n=s.shift(),t=parseInt(n,10);if(isNaN(t)||t===0)this.fg=null,this.bg=null,this.bold=!1,this.faint=!1,this.italic=!1,this.underline=!1;else if(t===1)this.bold=!0;else if(t===2)this.faint=!0;else if(t===3)this.italic=!0;else if(t===4)this.underline=!0;else if(t===21)this.bold=!1;else if(t===22)this.faint=!1,this.bold=!1;else if(t===23)this.italic=!1;else if(t===24)this.underline=!1;else if(t===39)this.fg=null;else if(t===49)this.bg=null;else if(t>=30&&t<38)this.fg=this.ansi_colors[0][t-30];else if(t>=40&&t<48)this.bg=this.ansi_colors[0][t-40];else if(t>=90&&t<98)this.fg=this.ansi_colors[1][t-90];else if(t>=100&&t<108)this.bg=this.ansi_colors[1][t-100];else if((t===38||t===48)&&s.length>0){let a=t===38,i=s.shift();if(i==="5"&&s.length>0){let o=parseInt(s.shift(),10);o>=0&&o<=255&&(a?this.fg=this.palette_256[o]:this.bg=this.palette_256[o])}if(i==="2"&&s.length>2){let o=parseInt(s.shift(),10),u=parseInt(s.shift(),10),b=parseInt(s.shift(),10);if(o>=0&&o<=255&&u>=0&&u<=255&&b>=0&&b<=255){let d={rgb:[o,u,b],class_name:"truecolor"};a?this.fg=d:this.bg=d}}}}}transform_to_html(e){let s=e.text;if(s.length===0||(s=this.escape_txt_for_html(s),!e.bold&&!e.italic&&!e.faint&&!e.underline&&e.fg===null&&e.bg===null))return s;let n=[],t=[],a=e.fg,i=e.bg;e.bold&&n.push(this._boldStyle),e.faint&&n.push(this._faintStyle),e.italic&&n.push(this._italicStyle),e.underline&&n.push(this._underlineStyle),this._use_classes?(a&&(a.class_name!=="truecolor"?t.push(`${a.class_name}-fg`):n.push(`color:rgb(${a.rgb.join(",")})`)),i&&(i.class_name!=="truecolor"?t.push(`${i.class_name}-bg`):n.push(`background-color:rgb(${i.rgb.join(",")})`))):(a&&n.push(`color:rgb(${a.rgb.join(",")})`),i&&n.push(`background-color:rgb(${i.rgb})`));let o="",u="";return t.length&&(o=` class="${t.join(" ")}"`),n.length&&(u=` style="${n.join(";")}"`),`<span${u}${o}>${s}</span>`}process_hyperlink(e){let s=e.url.split(":");return s.length<1||!this._url_allowlist[s[0]]?"":`<a href="${this.escape_txt_for_html(e.url)}">${this.escape_txt_for_html(e.text)}</a>`}}function O(r,...e){let s=r.raw[0],n=/^\s+|\s+\n|\s*#[\s\S]*?\n|\n/gm,t=s.replace(n,"");return new RegExp(t)}function Q(r,...e){let s=r.raw[0],n=/^\s+|\s+\n|\s*#[\s\S]*?\n|\n/gm,t=s.replace(n,"");return new RegExp(t,"g")}var B,I,j;const Y={class:"flex flex-col h-full"},Z={class:"px-6 py-4 border-b border-border flex items-center justify-between bg-bg-panel"},P={class:"flex items-center gap-4"},K={class:"font-mono text-[13px] text-fg-muted px-2 py-0.5 bg-bg-element rounded"},ee={key:1,class:"px-3.5 py-1.5 text-[13px] font-medium border border-success/40 rounded-md bg-success-muted text-success"},te={key:2,class:"px-3.5 py-1.5 text-[13px] font-medium border border-error/40 rounded-md bg-error-muted text-error"},se={class:"text-fg-subtle select-none text-right min-w-[28px] shrink-0"},ne=["innerHTML"],ie={key:0,class:"text-fg-muted"},le={key:0,class:"px-6 py-3 border-t border-border flex items-center justify-between bg-bg-panel"},re={class:"flex items-center gap-3 text-[13px]"},ae={class:"text-fg-muted font-mono text-xs"},oe=A({__name:"ExecutionView",props:{id:{}},setup(r){const e=r,s=W(),n=p([]),t=p("running"),a=p(""),i=p(null),o=p(Date.now()),u=p(null),b=new J;b.use_classes=!1;let d=null;const L=S(()=>`${(((u.value??Date.now())-o.value)/1e3).toFixed(1)}s`),R=S(()=>t.value!=="running"),$=S(()=>{const c=e.id.split("-");return c.length>1?c.slice(0,-1).join("-"):e.id});function U(){X(()=>{i.value&&(i.value.scrollTop=i.value.scrollHeight)})}D(()=>{o.value=Date.now(),d=H(e.id,c=>{n.value.push(c),U()},c=>{u.value=Date.now(),c.startsWith("error")?(t.value="error",a.value=c):(t.value="success",a.value="Completed successfully")},()=>{u.value=Date.now(),t.value="error",a.value="Connection lost"})}),N(()=>{d==null||d.close()});async function q(){await z(e.id)}function F(c){return b.ansi_to_html(c)}function w(){switch(t.value){case"running":return"bg-primary-muted text-primary";case"success":return"bg-success-muted text-success";case"error":return"bg-error-muted text-error"}}function E(){switch(t.value){case"running":return"Running";case"success":return"Completed";case"error":return"Failed"}}return(c,g)=>(_(),h("div",Y,[f("div",Z,[f("div",P,[g[1]||(g[1]=f("span",{class:"text-[15px] font-bold text-fg"},"Execution",-1)),f("span",K,x($.value),1),f("span",{class:m([w(),"inline-flex items-center gap-1.5 px-3 py-1 text-xs font-semibold rounded-full"])},[f("span",{class:m(["w-1.5 h-1.5 rounded-full bg-current",{"animate-pulse-dot":t.value==="running"}])},null,2),T(" "+x(E()),1)],2)]),t.value==="running"?(_(),h("button",{key:0,onClick:q,class:"px-3.5 py-1.5 text-[13px] font-medium border border-error rounded-md bg-transparent text-error cursor-pointer hover:bg-error-muted transition-all duration-150"}," Cancel ")):t.value==="success"?(_(),h("span",ee," Done ")):t.value==="error"?(_(),h("span",te," Execution failed ")):y("",!0)]),f("div",{ref_key:"outputEl",ref:i,class:"flex-1 overflow-y-auto px-6 py-4 bg-bg font-mono text-[13px] leading-[1.7]"},[(_(!0),h(G,null,V(n.value,(k,C)=>(_(),h("div",{key:C,class:"flex gap-3 whitespace-pre-wrap break-all"},[f("span",se,x(C+1),1),f("span",{class:"text-fg-muted",innerHTML:F(k)},null,8,ne)]))),128)),n.value.length===0&&t.value==="running"?(_(),h("div",ie," Waiting for output... ")):y("",!0)],512),R.value?(_(),h("div",le,[f("div",re,[f("span",{class:m([w(),"inline-flex items-center gap-1.5 px-3 py-1 text-xs font-semibold rounded-full"])},[g[2]||(g[2]=f("span",{class:"w-1.5 h-1.5 rounded-full bg-current"},null,-1)),T(" "+x(E()),1)],2),f("span",ae,x(L.value),1)]),f("button",{onClick:g[0]||(g[0]=k=>M(s).back()),class:"px-3.5 py-1.5 text-[13px] font-medium border border-border rounded-md bg-transparent text-fg-muted cursor-pointer hover:border-fg-subtle hover:text-fg transition-all duration-150"}," ← Back to runbook ")])):y("",!0)]))}});export{oe as default};
