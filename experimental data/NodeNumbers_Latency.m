%根据任务的变�? task 3500
x=[20,40,60,80,100];%x轴上的数据，第一个�?�代表数据开始，第二个�?�代表间隔，第三个�?�代表终�?
a=[2.97,3.97,4.86,7.89,13.29]; %a数据y�?
b=[2.59,6.66,15.72,31.78,48.7];
c=[2.7,7.46,19.78,36.12,56.4]; 
% yyaxis left   
plot(x,a,'-*b',x,b,'-^r',x,c,'-og','markersize',8,'linewidth',2); %线�?�，颜色，标�?
axis([18,105,0,72])  %确定x轴与y轴框图大�?
set(gca,'XTick',(20:20:100)) %x轴范�?1-6，间�?1
set(gca,'YTick',(0:10:70)) %y轴范�?0-700，间�?100
h=legend('BumbleBee','WaterBear-QS-Q','WaterBear-QS-C','Location','Best');  %右上角标�?
% set(h,'Box','off');
set(gca,'fontsize',12);
xlabel('�ڵ�����','fontsize',12) %x轴坐标描�?
ylabel('ʱ�� (s)','fontsize',12) %y轴坐标描�?
% yyaxis right
% plot(x,c,'--p',x,d,'-*b','markersize',10,'linewidth',2); %线�?�，颜色，标�?
% axis([0,8,0,3.2])  %确定x轴与y轴框图大�?
% set(gca,'XTick',x) %x轴范�?1-6，间�?1
% set(gca,'XTickLabel',{'1/2','1','2','4','8'}); 
% set(gca,'YTick',(0:0.4:3.2)) %y轴范�?0-700，间�?100
% h=legend('Packaging time','Single delay','Location','Best');  %右上角标�?
% %set(h,'Box','off');
% set(gca,'fontsize',18);
% ylabel('Time(s)','fontsize',18) %y轴坐标描�?