import { emit } from "brainkit";

export function run(): i32 {
  for (let i: i32 = 1; i <= 5; i++) {
    const topic = "as.test." + i.toString();
    const payload = '{"seq":' + i.toString() + '}';
    emit(topic, payload);
  }
  return 0;
}
