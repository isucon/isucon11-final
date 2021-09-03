local base = import './base.libsonnet';

base {
  arg_variant: 'contestant',
  //メモリを制限する
  //MEMO: 具体的な値は決めていないので一旦c5.largeデフォルトの2vcpu/4Gmemにしておく
  provisioners_plus:: [
    {
      type: 'shell',
      inline: [
        'sudo sh -c "echo GRUB_CMDLINE_LINUX=\"mem=2G\" > /etc/default/grub.d/99-isucon.cfg"',
        'sudo update-grub',
      ],
    },
  ],
}
