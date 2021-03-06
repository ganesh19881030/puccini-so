// This file was auto-generated from a YAML file

package v1_0

func init() {
	Profile["/hot/1.0/js/modulo.js"] = `

clout.exec('tosca.helpers');

function validate(v, rules) {
	if (arguments.length !== 2)
		throw 'must have 1 arguments';
	if ((rules.step === undefined) || (rules.offset === undefined))
		throw 'must provide "step" and "offset"';
	v = tosca.getComparable(v);
	var step = tosca.getComparable(rules.step);
	var offset = tosca.getComparable(rules.offset);
	return value % self.step == self.offset;
}
`
}
