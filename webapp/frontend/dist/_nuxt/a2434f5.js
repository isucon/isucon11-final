(window.webpackJsonp=window.webpackJsonp||[]).push([[15],{249:function(e,t,n){"use strict";n.r(t);var r=n(0).a.extend({name:"Pagination",props:{prevDisabled:{type:Boolean,required:!1,default:function(){return!1}},nextDisabled:{type:Boolean,required:!1,default:function(){return!1}}},computed:{prevClasses:function(){return this.getClasses(this.prevDisabled)},nextClasses:function(){return this.getClasses(this.nextDisabled)}},methods:{getClasses:function(e){return e?["text-gray-500"]:["cursor-pointer","text-black","hover:bg-primary-300","hover:text-white","hover:rounded"]}}}),l=n(20),component=Object(l.a)(r,(function(){var e=this,t=e.$createElement,n=e._self._c||t;return n("div",{staticClass:"flex flex-row items-center"},[n("div",{staticClass:"p-2 mr-6",class:e.prevClasses,on:{click:function(t){!e.prevDisabled&&e.$emit("goPrev")}}},[n("fa-icon",{staticClass:"mr-2",attrs:{icon:"chevron-left",size:"lg"}}),e._v(" "),n("span",{staticClass:"text-base"},[e._v(" Prev ")])],1),e._v(" "),n("div",{staticClass:"p-2",class:e.nextClasses,on:{click:function(t){!e.nextDisabled&&e.$emit("goNext")}}},[n("span",{staticClass:"text-base mr-2"},[e._v(" Next ")]),e._v(" "),n("fa-icon",{attrs:{icon:"chevron-right",size:"lg"}})],1)])}),[],!1,null,null,null);t.default=component.exports}}]);