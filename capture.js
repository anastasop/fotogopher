
var args = require('system').args;
var page = require('webpage').create();

page.clipRect = { top: 0, left: 0, width: 1280, height: 1024 };

//page.onError = function(msg) {
//	console.log('rendering failed: ' + msg);
//};

page.viewportSize = {
  width: 1280,
  height: 1024
};

page.open(args[1], function(status) {
	if (status === 'success') {
		if (args.length > 2) {
			page.render(args[2]);
		} else {
			console.log(page.renderBase64('JPEG'));
		}
		phantom.exit(0);
	} else {
		phantom.exit(1);
	}
});
